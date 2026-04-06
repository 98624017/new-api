package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	fixtureUsername    = "patchuser"
	fixturePassword    = "Password123"
	fixtureTokenKey    = "refundtasktoken"
	fixtureTaskID      = "task_refund_acceptance"
	fixtureChannelID   = 1
	fixtureUserID      = 1
	fixtureTokenID     = 1
	fixtureTaskQuota   = 200
	fixtureUserQuota   = 800
	fixtureTokenRemain = 300
	fixtureTokenUsed   = 200
)

func main() {
	mode := flag.String("mode", "seed", "seed or inspect")
	dbPath := flag.String("db", "", "sqlite database path")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "missing -db")
		os.Exit(1)
	}

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db failed: %v\n", err)
		os.Exit(1)
	}

	if err := ensureSchema(db); err != nil {
		fmt.Fprintf(os.Stderr, "ensure schema failed: %v\n", err)
		os.Exit(1)
	}

	switch *mode {
	case "seed":
		if err := seedFixture(db); err != nil {
			fmt.Fprintf(os.Stderr, "seed fixture failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("SEEDED_USER=%s\n", fixtureUsername)
		fmt.Printf("SEEDED_PASSWORD=%s\n", fixturePassword)
		fmt.Printf("SEEDED_TOKEN_KEY=sk-%s\n", fixtureTokenKey)
	case "inspect":
		if err := inspectFixture(db); err != nil {
			fmt.Fprintf(os.Stderr, "inspect fixture failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func ensureSchema(db *gorm.DB) error {
	common.UsingSQLite = true

	return db.AutoMigrate(
		&model.Setup{},
		&model.User{},
		&model.Token{},
		&model.Channel{},
		&model.Task{},
		&model.Log{},
	)
}

func seedFixture(db *gorm.DB) error {
	common.UsingSQLite = true

	if err := db.Exec("DELETE FROM tasks").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM tokens").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM users").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM logs").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM channels").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM setups").Error; err != nil {
		return err
	}

	now := time.Now().Unix()
	passwordHash, err := common.Password2Hash(fixturePassword)
	if err != nil {
		return err
	}

	setup := &model.Setup{
		ID:            1,
		Version:       "acceptance",
		InitializedAt: now,
	}
	if err := db.Create(setup).Error; err != nil {
		return err
	}

	user := &model.User{
		Id:          fixtureUserID,
		Username:    fixtureUsername,
		Password:    passwordHash,
		DisplayName: "Patch User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       fixtureUserQuota,
		AffCode:     "acc1",
	}
	if err := db.Create(user).Error; err != nil {
		return err
	}

	channel := &model.Channel{
		Id:     fixtureChannelID,
		Name:   "acceptance-channel",
		Key:    "dummy-key",
		Status: common.ChannelStatusEnabled,
		Group:  "default",
	}
	if err := db.Create(channel).Error; err != nil {
		return err
	}

	token := &model.Token{
		Id:          fixtureTokenID,
		UserId:      fixtureUserID,
		Key:         fixtureTokenKey,
		Name:        "acceptance-token",
		Status:      common.TokenStatusEnabled,
		RemainQuota: fixtureTokenRemain,
		UsedQuota:   fixtureTokenUsed,
		ExpiredTime: -1,
	}
	if err := db.Create(token).Error; err != nil {
		return err
	}

	task := &model.Task{
		TaskID:     fixtureTaskID,
		CreatedAt:  now - 120,
		UpdatedAt:  now - 120,
		UserId:     fixtureUserID,
		Group:      "default",
		ChannelId:  fixtureChannelID,
		Quota:      fixtureTaskQuota,
		Action:     "generate",
		Status:     model.TaskStatusInProgress,
		SubmitTime: now - 120,
		Progress:   "30%",
		Properties: model.Properties{
			OriginModelName: "test-model",
		},
		PrivateData: model.TaskPrivateData{
			BillingSource: "wallet",
			TokenId:       fixtureTokenID,
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.02,
				GroupRatio:      1.0,
				OriginModelName: "test-model",
			},
		},
	}
	return db.Create(task).Error
}

func inspectFixture(db *gorm.DB) error {
	var task model.Task
	if err := db.Where("task_id = ?", fixtureTaskID).First(&task).Error; err != nil {
		return err
	}

	var user model.User
	if err := db.Select("quota").Where("id = ?", fixtureUserID).First(&user).Error; err != nil {
		return err
	}

	var token model.Token
	if err := db.Select("remain_quota", "used_quota").Where("id = ?", fixtureTokenID).First(&token).Error; err != nil {
		return err
	}

	fmt.Printf("TASK_STATUS=%s\n", task.Status)
	fmt.Printf("TASK_FAIL_REASON=%s\n", task.FailReason)
	fmt.Printf("USER_QUOTA=%d\n", user.Quota)
	fmt.Printf("TOKEN_REMAIN=%d\n", token.RemainQuota)
	fmt.Printf("TOKEN_USED=%d\n", token.UsedQuota)
	return nil
}
