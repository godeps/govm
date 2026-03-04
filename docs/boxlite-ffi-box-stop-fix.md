# boxlite_ffi `box_stop` 句柄生命周期修复说明

## 背景
在 `govm` native 流程中，执行 `Start -> Stop -> Close` 会触发 SIGSEGV，栈落在 `govm_box_free -> boxlite_ffi::ops::box_free`。

典型症状：
- `box.Stop()` 返回成功
- `box.Close()`（底层 `box_free`）时进程崩溃

## 根因
上游实现（`other/boxlite/ffi/src/ops.rs`）的 `box_stop` 使用了：

```rust
let handle_ref = Box::from_raw(handle);
```

这会把 `handle` 的所有权取走，并在函数返回后 drop 掉 `BoxHandle`。随后调用者再执行 `box_free(handle)` 就是二次释放，导致崩溃。

正确行为应为“借用句柄”，而不是“消费句柄”。

## 上游修复建议
建议上游修复如下（当前未在上游仓库落地，按只读处理）：

- 目标文件：`ffi/src/ops.rs`
- 建议变更：

```diff
- let handle_ref = Box::from_raw(handle);
+ let handle_ref = &*handle;
```

## govm 侧兼容处理
在上游修复未发布前，`govm` bridge 已绕开该问题：
- `govm_box_stop` 不再调用 `boxlite_ffi::box_stop`，而是直接调用 `handle_ref.handle.stop()`（不消费裸指针）。

文件：
- `rust-bridge/src/lib.rs`

## 复现步骤（修复前）
1. 创建 box
2. `Start()`
3. `Stop()`
4. `Close()`
5. 触发 SIGSEGV

## 验证步骤（修复后）
1. 重新构建 bridge：
   - `make bridge-install-local`
2. 运行最小生命周期 smoke：
   - create/start/stop/close/remove
3. 运行离线示例：
   - `go run -tags govm_native ./examples/offline`
4. 预期：无崩溃，流程正常结束

## 风险评估
- 修复是生命周期语义纠正，不改变外部 API。
- 影响面集中在 FFI stop/free 路径。
- 建议上游补充回归测试：`stop + free` 不得崩溃。

## 建议上游提交内容
1. `ffi/src/ops.rs` 修复 `box_stop` 引用方式
2. 新增 FFI 生命周期回归测试（至少覆盖 create/attach/start/stop/free）
3. 在 changelog 标注此修复（memory safety / handle lifecycle）

## 回滚方案
如发现兼容问题，可临时在 `govm` 侧继续使用 bridge 绕过实现（当前已具备），并回滚上游 patch。
