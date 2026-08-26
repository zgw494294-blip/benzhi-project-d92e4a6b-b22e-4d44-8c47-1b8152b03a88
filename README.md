# 洁净工作台现场认证与启用放行服务

本项目为实验设施团队提供一条可追溯的洁净工作台现场认证流程。系统覆盖设备基线建档、认证方案锁定、有序测点实测、自动偏差判定、整改与定向复测、质量复核冻结、启用凭据签发及公开校验。所有修改均采用版本检查与幂等键保护，业务事件写入带摘要链的本地 JSON Lines 账本。

同一规范化工作台编号同时只允许一个在途案例；上一轮放行后再次建档会自动关联前序案例。方案锁定支持无落账预检，实测入口支持末次初测的完整纠错，偏差整改入口支持责任人与期限调整，待复核详情会返回带版本摘要的逐测点就绪清单，复核退回可在一次请求中原子提交多个测点问题。

## 构建与测试

```text
go build ./...
go test ./...
```

## 运行

服务默认只监听 `127.0.0.1:19081`，数据默认存放在当前目录的 `data` 子目录：

```text
go run ./cmd/server
```

可通过 `-addr=127.0.0.1:19100` 指定安全的回环监听地址；也可设置 `PORT`，此时绑定 `127.0.0.1:<PORT>`。显式地址不能使用未指定 IP、`0.0.0.0` 或非回环地址。

运行真实 HTTP 全流程自检并自动退出：

```text
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

## API 概览

所有写请求都要求 `Idempotency-Key` 请求头和 JSON 字段 `expectedVersion`（创建案例和方案预检除外）。主要入口位于 `/api/v1/certification-cases`，依次完成建档、方案锁定、测量、整改、复测、复核和凭据签发。

- `GET /api/v1/certification-cases?cabinetCode=...&status=...` 按规范化工作台编号和合法状态筛选案例。
- `POST /api/v1/certification-cases/{caseID}/plan/lock?preflight=true` 执行方案完整性预检，不写账本；也可在 JSON 中传入 `preflight: true`。
- `POST /api/v1/certification-cases/{caseID}/measurements` 传入 `correctsMeasurementID` 和 `correctionReason` 时更正当前末次初测。
- `POST /api/v1/certification-cases/{caseID}/deviations/remediate` 使用 `action: "adjust_assignment"` 及 `newAssignee`、`newDueAt`、`adjustmentReason`、`operator` 调整开放偏差责任。
- `GET /api/v1/certification-cases/{caseID}` 在待复核状态返回 `reviewReadiness`，并为未关闭偏差派生 `dueStatus` 和 `overdueDurationSeconds`。
- `POST /api/v1/certification-cases/{caseID}/review` 可在通过时传入 `checklistDigest`，或在退回时通过 `issues` 一次提交多个测点问题。

`GET /api/v1/credentials/{credentialID}/verify?code=...` 是只读公开校验接口，`GET /healthz` 用于健康检查。
