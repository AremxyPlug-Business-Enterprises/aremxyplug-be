package models

import "time"

const (
	TaskSignup            TaskType = "signup"
	TaskKYC               TaskType = "kyc"
	TaskFundWallet        TaskType = "fund_wallet"
	TaskTransactionVolume TaskType = "transaction_volume"
	TaskPointRedeem       TaskType = "point_redeem"
)

type TaskType string

// TaskEvent is the normalized event we use inside the tasks package.
type TaskEvent struct {
	Task   TaskType
	Amount int64 // for cumulative or one-time threshold check
	TxID   string
	Meta   map[string]interface{}
}

// TaskDef describes a task configuration.
type TaskDef struct {
	Code    TaskType
	OneTime bool
	Target  int64 // target amount (for cumulative or threshold)
}

// Task registry (static). Edit targets as required.
var TaskRegistry = map[TaskType]TaskDef{
	TaskSignup: {
		Code:    TaskSignup,
		OneTime: true,
		Target:  0,
	},
	TaskKYC: {
		Code:    TaskKYC,
		OneTime: true,
		Target:  0,
	},
	TaskFundWallet: {
		Code:    TaskFundWallet,
		OneTime: true,
		Target:  100, // currency smallest-unit expected
	},
	TaskTransactionVolume: {
		Code:    TaskTransactionVolume,
		OneTime: false,
		Target:  1000,
	},
	TaskPointRedeem: {
		Code:    TaskPointRedeem,
		OneTime: true,
		Target:  1,
	},
}

// ProgressDoc is the persisted task progress document.
type ProgressDoc struct {
	ID          string     `bson:"_id,omitempty" json:"id"`
	UserID      string     `bson:"user_id" json:"user_id"`
	TaskCode    TaskType   `bson:"task_code" json:"task_code"`
	Type        string     `bson:"type" json:"type"` // "one_time" | "cumulative"
	Progress    int64      `bson:"progress" json:"progress"`
	Target      int64      `bson:"target" json:"target"`
	Completed   bool       `bson:"completed" json:"completed"`
	CompletedAt *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
}
