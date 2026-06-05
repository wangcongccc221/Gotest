package database

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type EndProcessResult struct {
	CustomerID     int
	FruitName      string
	StartTime      string
	EndTime        string
	BatchNumber    uint64
	BatchWeight    float64
	CompletedState string
}

func EndCurrentFruitProcess(endAt time.Time) (EndProcessResult, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return EndProcessResult{}, err
	}
	if endAt.IsZero() {
		endAt = time.Now()
	}

	result := EndProcessResult{}
	if err := db.Transaction(func(tx *gorm.DB) error {
		fruit, hasFruit, err := realtimeSaveCurrentFruitInfo(tx)
		if err != nil {
			return err
		}
		if !hasFruit {
			return errors.New("no unfinished fruit process")
		}
		if strings.TrimSpace(fruit.StartTime) == "" {
			return errors.New("unfinished fruit process has empty StartTime")
		}

		endTime := endAt.Format("2006-01-02 15:04:05")
		if err := tx.Model(&TbFruitInfo{}).
			Where("CustomerID = ?", fruit.CustomerID).
			Updates(map[string]any{
				"EndTime":        endTime,
				"CompletedState": "1",
			}).Error; err != nil {
			return err
		}

		result = EndProcessResult{
			CustomerID:     fruit.CustomerID,
			FruitName:      fruit.FruitName,
			StartTime:      fruit.StartTime,
			EndTime:        endTime,
			CompletedState: "1",
		}
		if fruit.BatchNumber != nil {
			result.BatchNumber = *fruit.BatchNumber
		}
		if fruit.BatchWeight != nil {
			result.BatchWeight = *fruit.BatchWeight
		}
		return nil
	}); err != nil {
		return EndProcessResult{}, err
	}
	return result, nil
}
