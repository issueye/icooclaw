# Scheduler 模块优化方案

## 一、模块概述

该模块是一个定时任务调度器，负责管理和执行两类任务：
1. **Cron 任务** - 基于 Cron 表达式的定时任务
2. **立即执行任务** - 创建后立即执行一次

同时包含一个**心跳服务**，用于定期发送心跳事件和检查工作区状态。

### 文件结构

| 文件 | 功能 |
|------|------|
| scheduler.go | 主调度器，管理任务生命周期 |
| task.go | 定义任务接口和类型枚举 |
| cron_task.go | Cron 任务运行器 |
| once_task.go | 立即执行任务运行器 |
| heartbeat.go | 心跳服务 |
| cron_test.go | 单元测试 |

---

## 二、核心问题分析

### 问题 1：状态管理逻辑错误

**位置**：`cron_task.go:73-84` 和 `once_task.go:31-38`

**问题描述**：Run 方法先设置状态为 Running，然后才检查是否已经在运行，逻辑颠倒。

```go
// 当前错误代码
func (r *CronTaskRunner) Run(ctx context.Context) error {
    r.status = TaskRunning  // 先设置为 Running
    // ...
    if r.status == TaskRunning {  // 然后才检查，意义何在？
        r.logger.Info("任务已在运行中", "name", r.name)
        return nil
    }
}
```

**影响**：无法正确防止任务重复执行，状态判断形同虚设。

---

### 问题 2：并发安全问题

**位置**：`scheduler.go:125-144`

**问题描述**：`checkTasks` 在独立 goroutine 中运行，遍历 `TaskRunners` 时未加锁。

```go
// 当前代码
func (s *Scheduler) checkTasks() {
    for _, runner := range s.TaskRunners {  // 可能并发读写
        // ...
    }
}
```

**影响**：在高并发场景下可能导致 map 并发读写 panic。

---

### 问题 3：重复运行检查不准确

**位置**：`scheduler.go:137`

**问题描述**：使用硬编码 30 秒间隔判断重复执行。

```go
if now.Sub(runner.GetInfo().LastRunAt) > 30*time.Second {
    s.executeTask(runner)
}
```

**影响**：
- 无法适应不同频率的任务
- 应该使用任务自己的 nextRun 时间判断

---

### 问题 4：职责边界不清

**位置**：`cron_task.go:68-108`

**问题描述**：Runner 的 Run 方法内部既执行任务又判断是否应该运行。

```go
func (r *CronTaskRunner) Run(ctx context.Context) error {
    // 既要执行任务，又要判断是否应该执行
    if !r.ShouldRun(time.Now()) {
        // ...
    }
    // 执行任务
}
```

**影响**：调度器和 Runner 职责混淆，难以测试和维护。

---

### 问题 5：缺乏任务执行超时机制

**问题描述**：任务执行没有超时控制，可能导致任务永久阻塞。

**影响**：失控的任务会影响整个调度器稳定性。

---

### 问题 6：心跳服务类型安全问题

**位置**：`heartbeat.go:27`

```go
storage interface{}  // 使用空接口，不安全
```

---

## 三、优化方案

### 3.1 任务接口重构

**目标**：清晰定义职责边界，增加扩展性

```go
// task.go 重构

/**
 * TaskCallback 定义任务执行回调接口
 * 用于任务生命周期事件的外部处理
 */
type TaskCallback interface {
    OnStart(ctx context.Context, task *storage.Task)
    OnComplete(ctx context.Context, task *storage.Task, err error)
    OnNextRunCalculated(task *storage.Task, nextRun time.Time)
}

/**
 * TaskRunner 任务运行器接口
 * 职责：执行任务、管理自身状态、提供任务信息
 * 不负责判断是否应该执行（由调度器负责）
 */
type TaskRunner interface {
    Run(ctx context.Context) error
    Stop(ctx context.Context) error
    GetStatus() TaskRunStatus
    GetType() TaskType
    GetName() string
    GetInfo() *storage.Task
    ShouldRun(now time.Time) bool
    SetCallback(callback TaskCallback)
}
```

---

### 3.2 调度器核心重构

**目标**：解决并发安全、职责清晰、超时控制

```go
// scheduler.go 核心重构

/**
 * Scheduler 定时任务调度器
 * 职责：加载任务、调度执行、管理生命周期
 */
type Scheduler struct {
    bus         *bus.MessageBus
    storage     *storage.Storage
    config      *config.Config
    logger      *slog.Logger
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
    mu          sync.RWMutex
    TaskRunners map[string]TaskRunner
    running     bool
    taskCh      chan TaskRunner  // 任务执行队列
}

/**
 * run 调度器主循环
 * 使用单一 goroutine 处理所有任务调度，避免并发问题
 */
func (s *Scheduler) run() {
    defer s.wg.Done()

    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            s.checkTasks()
        case runner := <-s.taskCh:
            s.executeTask(runner)
        }
    }
}

/**
 * checkTasks 检查并调度应该执行的任务
 * 必须在锁保护下执行
 */
func (s *Scheduler) checkTasks() {
    s.mu.RLock()
    defer s.mu.RUnlock()

    now := time.Now()
    for _, runner := range s.TaskRunners {
        if !runner.GetInfo().Enabled {
            continue
        }
        // 使用 Runner 自己的 nextRun 判断
        if runner.ShouldRun(now) {
            select {
            case s.taskCh <- runner:
            default:
                // 队列满，忽略本次调度
            }
        }
    }
}

/**
 * executeTask 执行单个任务
 * 包含状态更新、超时控制、错误处理
 */
func (s *Scheduler) executeTask(runner TaskRunner) {
    info := runner.GetInfo()
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
    defer cancel()

    // 更新状态为 Running
    info.Status = TaskRunning
    info.LastRunAt = time.Now()
    _ = s.storage.UpdateTask(info)

    // 执行任务（带超时）
    err := runner.Run(ctx)

    // 执行完成后更新状态
    if err != nil {
        info.Status = TaskFailed
        s.logger.Error("任务运行失败", "name", info.Name, "error", err)
    } else {
        info.Status = TaskCompleted
        info.NextRunAt = runner.GetInfo().NextRunAt
    }

    _ = s.storage.UpdateTask(info)

    // 发送消息到消息总线
    s.publishTaskMessage(info, err)
}
```

---

### 3.3 CronTaskRunner 修复

**目标**：修正状态逻辑，职责单一

```go
// cron_task.go 重构

/**
 * CronTaskRunner Cron 任务运行器
 * 负责：根据 Cron 表达式执行任务，管理自身状态
 */
type CronTaskRunner struct {
    name       string
    task       *storage.Task
    cronExpr   string
    logger     *slog.Logger
    cronParser *CronParser
    lastRun    time.Time
    nextRun    time.Time
    status     TaskRunStatus
    taskType   TaskType
    callback   TaskCallback
    mu         sync.Mutex
}

/**
 * NewCronTaskRunner 创建 Cron 任务运行器
 */
func NewCronTaskRunner(name string, task *storage.Task, cronExpr string, logger *slog.Logger) (*CronTaskRunner, error) {
    parser := NewCronParser()
    if !parser.IsValid(cronExpr) {
        return nil, ErrInvalidCronExpression
    }

    now := time.Now()
    nextRun, ok := parser.NextRun(cronExpr, now)
    if !ok {
        return nil, ErrInvalidCronExpression
    }

    lastRun := task.LastRunAt
    if lastRun.IsZero() {
        lastRun = now
    }

    return &CronTaskRunner{
        name:       name,
        task:       task,
        cronExpr:   cronExpr,
        logger:     logger,
        cronParser: parser,
        status:     TaskPending,
        taskType:   TaskTypeCron,
        lastRun:    lastRun,
        nextRun:    nextRun,
    }, nil
}

/**
 * Run 执行任务
 * 只负责执行，不判断是否应该执行
 */
func (r *CronTaskRunner) Run(ctx context.Context) error {
    r.mu.Lock()
    if r.status == TaskRunning {
        r.mu.Unlock()
        return nil
    }
    r.status = TaskRunning
    r.mu.Unlock()

    r.logger.Info("任务开始执行", "name", r.name)

    // TODO: 执行实际任务逻辑

    // 更新状态和时间
    now := time.Now()
    nextRun, _ := r.cronParser.NextRun(r.cronExpr, now)
    r.nextRun = nextRun
    r.lastRun = now
    r.task.NextRunAt = nextRun
    r.task.LastRunAt = now

    r.mu.Lock()
    r.status = TaskCompleted
    r.mu.Unlock()

    r.logger.Info("任务执行完成", "name", r.name)
    return nil
}

/**
 * ShouldRun 判断是否应该执行
 * 基于下次运行时间判断
 */
func (r *CronTaskRunner) ShouldRun(now time.Time) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    return !r.nextRun.IsZero() && now.After(r.nextRun)
}

/**
 * SetCallback 设置任务回调
 */
func (r *CronTaskRunner) SetCallback(callback TaskCallback) {
    r.callback = callback
}
```

---

### 3.4 OnceTaskRunner 修复

```go
// once_task.go 重构

/**
 * OnceTaskRunner 立即执行任务运行器
 */
type OnceTaskRunner struct {
    name     string
    task     *storage.Task
    logger   *slog.Logger
    status   TaskRunStatus
    taskType TaskType
    mu       sync.Mutex
    callback TaskCallback
}

/**
 * ShouldRun 判断是否应该执行
 * 只运行一次，如果尚未运行且未完成则返回 true
 */
func (r *OnceTaskRunner) ShouldRun(now time.Time) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    return r.task.LastRunAt.IsZero() && r.status != TaskCompleted
}
```

---

### 3.5 心跳服务优化

```go
// heartbeat.go 重构

/**
 * Heartbeat 心跳服务
 */
type Heartbeat struct {
    bus     *bus.MessageBus
    storage *storage.Storage  // 使用具体类型
    logger  *slog.Logger
    config  *HeartbeatConfig
    wg      sync.WaitGroup
    ctx     context.Context
    cancel  context.CancelFunc
    running bool
    mu      sync.RWMutex
}
```

---

## 四、优化阶段与计划

### 第一阶段：基础修复（预计 1-2 天）

| 任务 | 描述 | 优先级 |
|------|------|--------|
| 修复状态管理逻辑 | 修正 CronTaskRunner 和 OnceTaskRunner 的状态检查顺序 | P0 |
| 添加并发保护 | 为 checkTasks 添加读写锁 | P0 |
| 修复重复执行判断 | 使用 nextRun 而非硬编码 30 秒 | P1 |

**具体工作**：
1. 修改 `cron_task.go` 的 `Run` 方法，将状态检查移到设置状态之前
2. 在 `scheduler.go` 的 `checkTasks` 方法中添加 `RLock`
3. 修改 `ShouldRun` 使用 `nextRun` 时间判断

---

### 第二阶段：架构优化（预计 2-3 天）

| 任务 | 描述 | 优先级 |
|------|------|--------|
| 重构任务接口 | 增加 TaskCallback 回调接口 | P1 |
| 引入执行队列 | 使用 channel 实现任务执行队列 | P1 |
| 添加超时控制 | 为任务执行添加 context 超时 | P1 |
| 类型安全修复 | 将 heartbeat 的 storage 改为具体类型 | P2 |

**具体工作**：
1. 在 `task.go` 中添加 `TaskCallback` 接口
2. 为 `Scheduler` 添加 `taskCh` channel
3. 在 `executeTask` 中使用 `context.WithTimeout`
4. 修改 `heartbeat.go` 的 storage 字段类型

---

### 第三阶段：功能增强（预计 2-3 天）

| 任务 | 描述 | 优先级 |
|------|------|--------|
| 重试机制 | 任务失败后自动重试 N 次 | P2 |
| 任务钩子 | 支持任务生命周期事件回调 | P2 |
| 动态调度 | 支持运行时添加/删除任务 | P2 |
| 指标监控 | 添加任务执行指标采集 | P3 |

**具体工作**：
1. 在任务运行器中添加重试计数器和重试逻辑
2. 实现 TaskCallback 接口，集成到调度器
3. 完善 AddTask/RemoveTask 的运行时处理
4. 添加 metrics 暴露任务执行统计

---

### 第四阶段：测试与文档（预计 1-2 天）

| 任务 | 描述 | 优先级 |
|------|------|--------|
| 单元测试补充 | 补充并发安全测试、边界测试 | P1 |
| 集成测试 | 添加调度器完整流程测试 | P2 |
| 文档完善 | 更新 API 文档和使用示例 | P2 |

---

## 五、实施优先级建议

```
P0 (必须修复):
├── 状态管理逻辑错误
└── 并发安全问题

P1 (应该修复):
├── 重复执行判断不准确
├── 职责边界不清
├── 超时控制缺失

P2 (建议优化):
├── 心跳服务类型安全
├── 任务执行队列
├── 回调机制

P3 (可选优化):
├── 重试机制
├── 指标监控
└── 动态调度
```

---

## 六、风险评估与注意事项

1. **向后兼容性**：重构可能影响现有 API，需要做好接口兼容
2. **测试覆盖**：重构期间需要确保现有测试通过
3. **灰度发布**：建议先在测试环境验证，再部署到生产环境
4. **日志保留**：重构过程中保留足够的日志便于问题排查

---

## 七、预期收益

| 指标 | 当前状态 | 优化后预期 |
|------|----------|------------|
| 并发安全 | 存在风险 | 完全安全 |
| 任务重复执行 | 可能发生 | 完全避免 |
| 失控任务处理 | 无超时 | 30 分钟超时 |
| 状态一致性 | 可能不一致 | 强一致性 |
| 扩展性 | 一般 | 良好（回调、队列） |

---

*文档生成时间：2026-03-02*
