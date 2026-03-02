# Scheduler 模块

定时任务调度器模块，提供 Cron 任务和一次性任务的调度执行能力。

## 功能特性

- **Cron 任务**：支持标准 Cron 表达式定时执行任务
- **一次性任务**：立即执行一次的任务
- **任务重试**：支持配置重试次数、间隔和退避策略
- **执行指标**：提供任务执行统计（成功/失败次数、平均耗时等）
- **回调机制**：任务开始、完成、下次运行时间计算的回调通知
- **心跳服务**：支持心跳检测功能

## 核心组件

### 接口定义

| 接口 | 说明 |
|------|------|
| `TaskRunner` | 任务运行器接口 |
| `TaskExecutor` | 任务执行器接口 |
| `TaskStorage` | 任务存储接口 |
| `TaskCallback` | 任务回调接口 |
| `SchedulerConfig` | 调度器配置接口 |
| `Logger` | 日志接口 |

### 任务类型

```go
const (
    TaskTypeCron     TaskType = 0  // Cron 任务
    TaskTypeInterval TaskType = 1  // 间隔任务
    TaskTypeOnce     TaskType = 2  // 一次性任务
)
```

### 任务状态

```go
const (
    TaskPending    TaskRunStatus = 0  // 待运行
    TaskRunning    TaskRunStatus = 1  // 运行中
    TaskCompleted  TaskRunStatus = 2  // 已完成
    TaskTerminated TaskRunStatus = 3  // 已终止
    TaskFailed     TaskRunStatus = 4  // 运行失败
    TaskRetrying   TaskRunStatus = 5  // 重试中
)
```

## 使用示例

### 创建调度器

```go
// 创建配置
config := NewDefaultSchedulerConfig()

// 创建存储（实现 TaskStorage 接口）
storage := NewMockTaskStorage(tasks)

// 创建调度器
scheduler := NewSchedulerWithSlog(storage, config, logger)

// 启动调度器
err := scheduler.Start()
if err != nil {
    log.Fatal(err)
}

// 停止调度器
defer scheduler.Stop()
```

### 创建 Cron 任务

```go
task := &TaskInfo{
    ID:       1,
    Name:     "my-cron-task",
    Type:     TaskTypeCron,
    CronExpr: "*/5 * * * *", // 每5分钟执行
    Enabled:  true,
}

runner, err := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)
```

### 创建一次性任务

```go
task := &TaskInfo{
    ID:      1,
    Name:    "my-once-task",
    Type:    TaskTypeOnce,
    Enabled: true,
}

runner, err := NewOnceTaskRunner(task.Name, task, logger)
```

### 配置重试

```go
config := RetryConfig{
    MaxRetries:  3,
    Interval:    time.Second * 5,
    Backoff:     2.0,
    MaxInterval: time.Minute,
}

runner.SetRetryConfig(config)
```

### 设置回调

```go
type MyCallback struct{}

func (m *MyCallback) OnStart(ctx context.Context, task *TaskInfo) {
    fmt.Printf("任务开始: %s\n", task.Name)
}

func (m *MyCallback) OnComplete(ctx context.Context, task *TaskInfo, err error) {
    if err != nil {
        fmt.Printf("任务失败: %s, error: %v\n", task.Name, err)
    } else {
        fmt.Printf("任务完成: %s\n", task.Name)
    }
}

func (m *MyCallback) OnNextRunCalculated(task *TaskInfo, nextRun time.Time) {
    fmt.Printf("下次运行: %s at %s\n", task.Name, nextRun)
}

runner.SetCallback(&MyCallback{})
```

## Cron 表达式

模块使用 `gronx` 库解析 Cron 表达式，支持 5 字段格式：

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日期 (1 - 31)
│ │ │ ┌───────────── 月份 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6)
│ │ │ │ │
* * * * *
```

### 示例

| 表达式 | 说明 |
|--------|------|
| `* * * * *` | 每分钟 |
| `*/5 * * * *` | 每 5 分钟 |
| `0 * * * *` | 每小时整点 |
| `0 0 * * *` | 每天午夜 |
| `0 0 * * 0` | 每周日午夜 |
| `0 0 1 * *` | 每月 1 日午夜 |

## 运行测试

```bash
cd agent/scheduler
go test -v ./...
```

## 文件结构

```
scheduler/
├── cron_task.go        # Cron 任务运行器
├── cron_task_test.go   # Cron 任务测试
├── errors.go           # 错误定义
├── heartbeat.go        # 心跳服务
├── interfaces.go       # 接口定义
├── once_task.go        # 一次性任务运行器
├── once_task_test.go   # 一次性任务测试
├── scheduler.go        # 调度器核心
├── scheduler_test.go   # 调度器测试
├── task.go             # 任务定义和接口
└── task_test.go        # 任务测试
```
