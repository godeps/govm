use std::collections::HashMap;
use std::ffi::{c_void, CStr, CString};
use std::os::raw::{c_char, c_int};
use std::path::PathBuf;
use std::ptr;
use std::time::Duration;

use boxlite::runtime::options::{
    BoxOptions as RuntimeBoxOptions, NetworkSpec, PortProtocol, PortSpec, RootfsSpec,
};
use boxlite::BoxCommand;
use boxlite::BoxliteOptions;
use boxlite::BoxliteRuntime;
use boxlite_ffi::error::{BoxliteErrorCode, FFIError};
use boxlite_ffi::ops::{
    box_attach, box_free, box_id, box_inspect_handle, box_list, box_remove, box_start, error_free,
    OutputCallback,
};
use boxlite_ffi::runtime::{create_tokio_runtime, BoxHandle, RuntimeHandle};
use futures::StreamExt;

fn set_error(out_err: *mut *mut c_char, msg: &str) {
    if !out_err.is_null() {
        if let Ok(c_msg) = CString::new(msg.to_string()) {
            unsafe {
                *out_err = c_msg.into_raw();
            }
        }
    }
}

unsafe fn c_str_to_string(s: *const c_char) -> Result<String, String> {
    if s.is_null() {
        return Err("null pointer".to_string());
    }
    CStr::from_ptr(s)
        .to_str()
        .map(|v| v.to_string())
        .map_err(|e| format!("invalid UTF-8: {e}"))
}

fn error_msg(err: &FFIError) -> String {
    if err.message.is_null() {
        "unknown error".to_string()
    } else {
        unsafe { CStr::from_ptr(err.message) }
            .to_str()
            .unwrap_or("unknown error")
            .to_string()
    }
}

#[no_mangle]
pub unsafe extern "C" fn govm_free_string(s: *mut c_char) {
    if !s.is_null() {
        drop(CString::from_raw(s));
    }
}

#[no_mangle]
pub unsafe extern "C" fn govm_runtime_new(
    config_json: *const c_char,
    out_err: *mut *mut c_char,
) -> *mut RuntimeHandle {
    let mut options = BoxliteOptions::default();

    if !config_json.is_null() {
        let config_str = match c_str_to_string(config_json) {
            Ok(s) => s,
            Err(e) => {
                set_error(out_err, &format!("invalid config: {e}"));
                return ptr::null_mut();
            }
        };

        if let Ok(json) = serde_json::from_str::<serde_json::Value>(&config_str) {
            if let Some(home_dir) = json.get("home_dir").and_then(|v| v.as_str()) {
                options.home_dir = PathBuf::from(home_dir);
            }
            if let Some(registries) = json.get("image_registries").and_then(|v| v.as_array()) {
                options.image_registries = registries
                    .iter()
                    .filter_map(|v| v.as_str().map(String::from))
                    .collect();
            }
        }
    }

    let tokio_rt = match create_tokio_runtime() {
        Ok(rt) => rt,
        Err(e) => {
            set_error(out_err, &format!("failed to create tokio runtime: {e}"));
            return ptr::null_mut();
        }
    };

    let runtime = match BoxliteRuntime::new(options) {
        Ok(rt) => rt,
        Err(e) => {
            set_error(out_err, &format!("failed to create runtime: {e}"));
            return ptr::null_mut();
        }
    };

    Box::into_raw(Box::new(RuntimeHandle { runtime, tokio_rt }))
}

#[no_mangle]
pub unsafe extern "C" fn govm_runtime_free(runtime: *mut RuntimeHandle) {
    if !runtime.is_null() {
        drop(Box::from_raw(runtime));
    }
}

#[no_mangle]
pub unsafe extern "C" fn govm_create_box(
    runtime: *mut RuntimeHandle,
    opts_json: *const c_char,
    name: *const c_char,
    out_err: *mut *mut c_char,
) -> *mut c_char {
    #[derive(serde::Deserialize)]
    struct GovmPortForward {
        host_ip: Option<String>,
        host_port: Option<u16>,
        guest_port: Option<u16>,
        protocol: Option<String>,
    }

    #[derive(serde::Deserialize)]
    struct GovmBoxOptionsJson {
        image: Option<String>,
        local_bundle_path: Option<String>,
        rootfs_path: Option<String>,
        cpus: Option<u8>,
        memory_mb: Option<u32>,
        env: Option<HashMap<String, String>>,
        working_dir: Option<String>,
        network_mode: Option<String>,
        #[allow(dead_code)]
        network_policy_mode: Option<String>,
        port_forwards: Option<Vec<GovmPortForward>>,
        macos_network_enabled: Option<bool>,
        auto_remove: Option<bool>,
        detach: Option<bool>,
    }

    if runtime.is_null() {
        set_error(out_err, "runtime is null");
        return ptr::null_mut();
    }
    if opts_json.is_null() {
        set_error(out_err, "opts_json is null");
        return ptr::null_mut();
    }

    let runtime_ref = &mut *runtime;

    let opts_str = match c_str_to_string(opts_json) {
        Ok(s) => s,
        Err(e) => {
            set_error(out_err, &format!("invalid opts_json: {e}"));
            return ptr::null_mut();
        }
    };

    let govm_opts: GovmBoxOptionsJson = match serde_json::from_str(&opts_str) {
        Ok(v) => v,
        Err(e) => {
            set_error(out_err, &format!("invalid opts_json: {e}"));
            return ptr::null_mut();
        }
    };

    let bundle_path = govm_opts.local_bundle_path.or(govm_opts.rootfs_path);

    let rootfs = if let Some(path) = bundle_path {
        RootfsSpec::RootfsPath(path)
    } else {
        RootfsSpec::Image(
            govm_opts
                .image
                .unwrap_or_else(|| "python:3.12-alpine".to_string()),
        )
    };

    let mut box_options = RuntimeBoxOptions {
        cpus: govm_opts.cpus,
        memory_mib: govm_opts.memory_mb,
        working_dir: govm_opts.working_dir,
        rootfs,
        auto_remove: govm_opts.auto_remove.unwrap_or(false),
        detach: govm_opts.detach.unwrap_or(false),
        ..Default::default()
    };

    if let Some(env) = govm_opts.env {
        box_options.env = env.into_iter().collect();
    }

    let network_mode = govm_opts
        .network_mode
        .clone()
        .unwrap_or_else(|| "nat".to_string())
        .to_lowercase();
    match network_mode.as_str() {
        "nat" => {
            box_options.network = NetworkSpec::Isolated;
        }
        "disabled" => {
            box_options.network = NetworkSpec::Isolated;
            box_options.advanced.security.network_enabled =
                govm_opts.macos_network_enabled.unwrap_or(false);
        }
        "bridged" => {
            set_error(
                out_err,
                "network_mode=bridged is not supported by current boxlite backend",
            );
            return ptr::null_mut();
        }
        _ => {
            set_error(out_err, &format!("unsupported network_mode={network_mode}"));
            return ptr::null_mut();
        }
    }

    if let Some(enabled) = govm_opts.macos_network_enabled {
        box_options.advanced.security.network_enabled = enabled;
    }

    if let Some(forwards) = govm_opts.port_forwards {
        if network_mode == "disabled" && !forwards.is_empty() {
            set_error(out_err, "network disabled cannot publish ports");
            return ptr::null_mut();
        }
        let mut ports = Vec::with_capacity(forwards.len());
        for (idx, f) in forwards.into_iter().enumerate() {
            let guest_port = match f.guest_port {
                Some(p) if p > 0 => p,
                _ => {
                    set_error(out_err, &format!("port_forwards[{idx}] missing guest_port"));
                    return ptr::null_mut();
                }
            };
            let protocol = match f
                .protocol
                .unwrap_or_else(|| "tcp".to_string())
                .to_lowercase()
                .as_str()
            {
                "tcp" => PortProtocol::Tcp,
                "udp" => PortProtocol::Udp,
                v => {
                    set_error(
                        out_err,
                        &format!("port_forwards[{idx}] invalid protocol={v}, expect tcp/udp"),
                    );
                    return ptr::null_mut();
                }
            };
            ports.push(PortSpec {
                host_port: f.host_port,
                guest_port,
                protocol,
                host_ip: f.host_ip,
            });
        }
        box_options.ports = ports;
    }

    let name_opt = if name.is_null() {
        None
    } else {
        match c_str_to_string(name) {
            Ok(s) if !s.is_empty() => Some(s),
            Ok(_) => None,
            Err(e) => {
                set_error(out_err, &format!("invalid name: {e}"));
                return ptr::null_mut();
            }
        }
    };

    let result = runtime_ref
        .tokio_rt
        .block_on(runtime_ref.runtime.create(box_options, name_opt));

    match result {
        Ok(handle) => {
            let box_id_val = handle.id().clone();
            let box_handle = Box::new(BoxHandle {
                handle,
                box_id: box_id_val,
                tokio_rt: runtime_ref.tokio_rt.clone(),
            });
            let raw = Box::into_raw(box_handle);
            let id = box_id(raw);
            box_free(raw);
            id
        }
        Err(e) => {
            set_error(out_err, &e.to_string());
            ptr::null_mut()
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn govm_get_box(
    runtime: *mut RuntimeHandle,
    id_or_name: *const c_char,
    out_err: *mut *mut c_char,
) -> *mut BoxHandle {
    let mut error = FFIError::default();
    let mut handle: *mut BoxHandle = ptr::null_mut();

    let code = box_attach(runtime, id_or_name, &mut handle, &mut error);
    if code == BoxliteErrorCode::NotFound {
        error_free(&mut error);
        return ptr::null_mut();
    }
    if code != BoxliteErrorCode::Ok {
        let msg = error_msg(&error);
        error_free(&mut error);
        set_error(out_err, &msg);
        return ptr::null_mut();
    }

    handle
}

#[no_mangle]
pub unsafe extern "C" fn govm_list_boxes(
    runtime: *mut RuntimeHandle,
    out_json: *mut *mut c_char,
    out_err: *mut *mut c_char,
) -> c_int {
    let mut error = FFIError::default();
    let code = box_list(runtime, out_json, &mut error);
    if code != BoxliteErrorCode::Ok {
        let msg = error_msg(&error);
        error_free(&mut error);
        set_error(out_err, &msg);
        return -1;
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn govm_remove_box(
    runtime: *mut RuntimeHandle,
    id_or_name: *const c_char,
    force: bool,
    out_err: *mut *mut c_char,
) -> c_int {
    let mut error = FFIError::default();
    let code = box_remove(runtime, id_or_name, force, &mut error);
    if code != BoxliteErrorCode::Ok {
        let msg = error_msg(&error);
        error_free(&mut error);
        set_error(out_err, &msg);
        return -1;
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_start(
    handle: *mut BoxHandle,
    out_err: *mut *mut c_char,
) -> c_int {
    let mut error = FFIError::default();
    let code = box_start(handle, &mut error);
    if code != BoxliteErrorCode::Ok {
        let msg = error_msg(&error);
        error_free(&mut error);
        set_error(out_err, &msg);
        return -1;
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_stop(handle: *mut BoxHandle, out_err: *mut *mut c_char) -> c_int {
    if handle.is_null() {
        set_error(out_err, "handle is null");
        return -1;
    }

    let handle_ref = &*handle;
    let result = handle_ref.tokio_rt.block_on(handle_ref.handle.stop());
    if let Err(e) = result {
        set_error(out_err, &e.to_string());
        return -1;
    }

    0
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_info(
    handle: *mut BoxHandle,
    out_json: *mut *mut c_char,
    out_err: *mut *mut c_char,
) -> c_int {
    let mut error = FFIError::default();
    let code = box_inspect_handle(handle, out_json, &mut error);
    if code != BoxliteErrorCode::Ok {
        let msg = error_msg(&error);
        error_free(&mut error);
        set_error(out_err, &msg);
        return -1;
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_id(handle: *mut BoxHandle) -> *mut c_char {
    if handle.is_null() {
        return ptr::null_mut();
    }
    box_id(handle)
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_free(handle: *mut BoxHandle) {
    if !handle.is_null() {
        box_free(handle);
    }
}

#[derive(serde::Deserialize, Default)]
struct ExecOptsJson {
    #[serde(default)]
    args: Vec<String>,
    #[serde(default)]
    env: Option<HashMap<String, String>>,
    #[serde(default)]
    tty: bool,
    #[serde(default)]
    user: Option<String>,
    #[serde(default)]
    timeout_secs: Option<f64>,
    #[serde(default)]
    working_dir: Option<String>,
}

#[no_mangle]
pub unsafe extern "C" fn govm_box_exec(
    handle: *mut BoxHandle,
    command: *const c_char,
    opts_json: *const c_char,
    callback: Option<OutputCallback>,
    user_data: *mut c_void,
    out_exit_code: *mut c_int,
    out_err: *mut *mut c_char,
) -> c_int {
    if handle.is_null() {
        set_error(out_err, "handle is null");
        return -1;
    }
    if out_exit_code.is_null() {
        set_error(out_err, "out_exit_code is null");
        return -1;
    }

    let handle_ref = &mut *handle;

    let cmd_str = match c_str_to_string(command) {
        Ok(s) => s,
        Err(e) => {
            set_error(out_err, &format!("invalid command: {e}"));
            return -1;
        }
    };

    let opts: ExecOptsJson = if !opts_json.is_null() {
        match c_str_to_string(opts_json) {
            Ok(json_str) => match serde_json::from_str(&json_str) {
                Ok(o) => o,
                Err(e) => {
                    set_error(out_err, &format!("invalid opts_json: {e}"));
                    return -1;
                }
            },
            Err(e) => {
                set_error(out_err, &format!("invalid opts_json string: {e}"));
                return -1;
            }
        }
    } else {
        ExecOptsJson::default()
    };

    let mut cmd = BoxCommand::new(&cmd_str).args(opts.args).tty(opts.tty);

    if let Some(env_map) = opts.env {
        for (k, v) in env_map {
            cmd = cmd.env(k, v);
        }
    }
    if let Some(user) = opts.user {
        cmd = cmd.user(user);
    }
    if let Some(secs) = opts.timeout_secs {
        cmd = cmd.timeout(Duration::from_secs_f64(secs));
    }
    if let Some(dir) = opts.working_dir {
        cmd = cmd.working_dir(dir);
    }

    let result = handle_ref.tokio_rt.block_on(async {
        let mut execution = handle_ref.handle.exec(cmd).await?;

        if let Some(cb) = callback {
            let mut stdout = execution.stdout();
            let mut stderr = execution.stderr();

            loop {
                tokio::select! {
                    Some(line) = async {
                        match &mut stdout {
                            Some(s) => s.next().await,
                            None => None,
                        }
                    } => {
                        let c_text = CString::new(line).unwrap_or_default();
                        cb(c_text.as_ptr(), 0, user_data);
                    }
                    Some(line) = async {
                        match &mut stderr {
                            Some(s) => s.next().await,
                            None => None,
                        }
                    } => {
                        let c_text = CString::new(line).unwrap_or_default();
                        cb(c_text.as_ptr(), 1, user_data);
                    }
                    else => break,
                }
            }
        }

        let status = execution.wait().await?;
        Ok::<i32, boxlite::BoxliteError>(status.exit_code)
    });

    match result {
        Ok(exit_code) => {
            *out_exit_code = exit_code;
            0
        }
        Err(e) => {
            set_error(out_err, &e.to_string());
            -1
        }
    }
}
