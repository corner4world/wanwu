package orm

import (
	"context"
	"encoding/json"
	"errors"

	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/model-service/client/model"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"gorm.io/gorm"
)

type Client struct {
	db *gorm.DB
}

func NewClient(ctx context.Context, db *gorm.DB) (*Client, error) {
	// auto migrate
	if err := db.AutoMigrate(
		model.ModelImported{},
		model.ModelExperienceDialog{},
		model.ModelExperienceDialogRecord{},
	); err != nil {
		return nil, err
	}

	// 数据初始化处理
	if err := initModelImportedProviderName(db); err != nil {
		return nil, err
	}
	if err := initModelUUID(db); err != nil {
		return nil, err
	}
	if err := initLLMDefaultFields(db); err != nil {
		return nil, err
	}
	return &Client{
		db: db,
	}, nil
}

func toErrStatus(key string, args ...string) *err_code.Status {
	return &err_code.Status{
		TextKey: key,
		Args:    args,
	}
}

const (
	providerHuoshanOriginal = "Huoshan" // 原始值
	providerHuoshanTarget   = "HuoShan" // 目标值
)

// initModelImportedProviderName 更新数据库中ModelImported表的provider字段值
func initModelImportedProviderName(dbClient *gorm.DB) error {
	err := dbClient.Model(&model.ModelImported{}).
		Where("provider = ?", providerHuoshanOriginal).
		Update("provider", providerHuoshanTarget).Error
	if err != nil {
		return err
	}
	return nil
}

// initModelUUID 批量更新数据库中ModelImported表的uuid字段值
func initModelUUID(dbClient *gorm.DB) error {
	const batchSize = 100

	for {
		var ids []uint32
		if err := dbClient.Model(&model.ModelImported{}).Select("id").Where("uuid = ? OR uuid IS NULL", "").Limit(batchSize).Find(&ids).Error; err != nil {
			return err
		}

		if len(ids) == 0 {
			break
		}

		caseWhen := "CASE id "
		var args []interface{}
		for _, id := range ids {
			caseWhen += "WHEN ? THEN ? "
			args = append(args, id, util.NewID())
		}
		caseWhen += "END"

		if err := dbClient.Model(&model.ModelImported{}).
			Where("id IN ?", ids).
			UpdateColumn("uuid", gorm.Expr(caseWhen, args...)).Error; err != nil {
			log.Errorf("init model uuid batch update error: %v", err)
			return err
		}
	}

	return nil
}

func initLLMDefaultFields(dbClient *gorm.DB) error {
	const batchSize = 100
	offset := 0

	for {
		var models []model.ModelImported
		if err := dbClient.Where("model_type = ?", "llm").Offset(offset).Limit(batchSize).Find(&models).Error; err != nil {
			return err
		}

		if len(models) == 0 {
			break
		}

		for _, m := range models {
			var cfg map[string]interface{}
			if err := json.Unmarshal([]byte(m.ProviderConfig), &cfg); err != nil {
				log.Errorf("unmarshal model config error: %v, id: %d", err, m.ID)
				continue
			}

			needsUpdate := false
			fields := []string{"functionCalling", "visionSupport", "thinkingSupport"}
			for _, f := range fields {
				if v, ok := cfg[f]; !ok || v == "" {
					cfg[f] = "noSupport"
					needsUpdate = true
				}
			}

			if needsUpdate {
				newCfg, err := json.Marshal(cfg)
				if err != nil {
					log.Errorf("marshal model config error: %v, id: %d", err, m.ID)
					continue
				}
				if err := dbClient.Model(&model.ModelImported{}).Where("id = ?", m.ID).Update("provider_config", string(newCfg)).Error; err != nil {
					log.Errorf("update model config error: %v, id: %d", err, m.ID)
				}
			}
		}

		offset += batchSize
	}

	return nil
}

func (c *Client) transaction(ctx context.Context, fc func(tx *gorm.DB) *err_code.Status) *err_code.Status {
	var status *err_code.Status
	_ = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if status = fc(tx); status != nil {
			return errors.New(status.String())
		}
		return nil
	})
	return status
}
