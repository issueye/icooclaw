package storage

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskType int

const (
	TaskTypeImmediate TaskType = iota // 立即执行任务
	TaskTypeCron                      // cron 任务
)

func (t TaskType) String() string {
	switch t {
	case TaskTypeImmediate:
		return "immediate"
	case TaskTypeCron:
		return "cron"
	default:
		return "unknown"
	}
}

// Task 定时任务模型
type Task struct {
	Model                 // 嵌入 Model 结构体
	Name        string    `gorm:"size:100;uniqueIndex" json:"name"`
	Description string    `gorm:"size:500" json:"description"` // 任务描述
	Type        TaskType  `gorm:"default:0" json:"type"`       // 任务类型 0 立即执行任务， 1 cron 任务
	CronExpr    string    `gorm:"size:100" json:"cron_expr"`   // Cron表达式
	Interval    int       `gorm:"default:0" json:"interval"`   // 固定间隔(秒)
	Message     string    `gorm:"type:text" json:"message"`    // 触发消息
	Channel     string    `gorm:"size:50" json:"channel"`      // 投递通道
	ChatID      string    `gorm:"size:255" json:"chat_id"`     // 投递目标
	Enabled     bool      `gorm:"default:true" json:"enabled"` // 是否启用
	NextRunAt   time.Time `gorm:"index" json:"next_run_at"`    // 下次执行时间
	LastRunAt   time.Time `json:"last_run_at"`                 // 上次执行时间
}

// TableName 表名
func (Task) TableName() string {
	return tableNamePrefix + "tasks"
}

// BeforeCreate 创建前回调
func (c *Task) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// TaskStorage 任务存储
type TaskStorage struct {
	db *gorm.DB
}

// NewTaskStorage 创建任务存储
func NewTaskStorage(db *gorm.DB) *TaskStorage {
	return &TaskStorage{db: db}
}

// CreateOrUpdate 创建或更新任务
func (s *TaskStorage) CreateOrUpdate(task *Task) error {
	return s.db.Save(task).Error
}

// Create 创建任务
func (s *TaskStorage) Create(task *Task) error {
	return s.db.Create(task).Error
}

// Update 更新任务
func (s *TaskStorage) Update(task *Task) error {
	return s.db.Save(task).Error
}

// GetByID 通过ID获取任务
func (s *TaskStorage) GetByID(id string) (*Task, error) {
	var task Task
	err := s.db.First(&task, id).Error
	return &task, err
}

// GetByName 通过名称获取任务
func (s *TaskStorage) GetByName(name string) (*Task, error) {
	var task Task
	err := s.db.Where("name = ?", name).First(&task).Error
	return &task, err
}

// Delete 删除任务
func (s *TaskStorage) Delete(id string) error {
	return s.db.Delete(&Task{}, id).Error
}

// GetAll 获取所有任务
func (s *TaskStorage) GetAll() ([]Task, error) {
	var tasks []Task
	err := s.db.Find(&tasks).Error
	return tasks, err
}

// GetEnabled 获取启用的任务
func (s *TaskStorage) GetEnabled() ([]Task, error) {
	var tasks []Task
	err := s.db.Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// GetDue 获取到期的任务
func (s *TaskStorage) GetDue() ([]Task, error) {
	var tasks []Task
	err := s.db.
		Where("enabled = ? AND (next_run_at IS NULL OR next_run_at <= ?)", true, time.Now()).
		Find(&tasks).Error
	return tasks, err
}

// UpdateNextRun 更新下次执行时间
func (s *TaskStorage) UpdateNextRun(id string, nextRun time.Time) error {
	return s.db.Model(&Task{}).Where("id = ?", id).Update("next_run_at", nextRun).Error
}

// UpdateLastRun 更新上次执行时间
func (s *TaskStorage) UpdateLastRun(id string) error {
	return s.db.Model(&Task{}).Where("id = ?", id).Update("last_run_at", time.Now()).Error
}

// ToggleEnabled 切换启用状态
func (s *TaskStorage) ToggleEnabled(id string) error {
	return s.db.Model(&Task{}).Where("id = ?", id).Update("enabled", gorm.Expr("NOT enabled")).Error
}

// Page 分页获取任务
func (s *TaskStorage) Page(q *QueryTask) (*ResQueryTask, error) {
	var total int64
	query := s.db.Model(&Task{})
	if q.KeyWord != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var tasks []Task
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&tasks).Error

	q.Page.Total = int(total)
	return &ResQueryTask{
		Page:    q.Page,
		Records: tasks,
	}, err
}

// QueryTask 任务查询参数
type QueryTask struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

// ResQueryTask 任务查询结果
type ResQueryTask struct {
	Page    Page   `json:"page"`
	Records []Task `json:"records"`
}
