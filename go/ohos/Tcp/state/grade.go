package state

import (
	"sync"

	"gotest/ohos/Tcp/protocol"
)

var gradeState = struct {
	sync.RWMutex
	latestGrade        protocol.StGradeInfo
	hasLatestGrade     bool
	latestExitInfos    protocol.StExitInfos
	latestExitInfosFSM int32
	hasLatestExitInfos bool
	latestMotor        protocol.StMotorInfo
	latestMotorFSM     int32
	hasLatestMotor     bool
	gradeInfoByDestID  map[int32]protocol.StGradeInfo
}{
	gradeInfoByDestID: make(map[int32]protocol.StGradeInfo),
}

func SetLatestGradeInfoSnapshot(grade protocol.StGradeInfo) {
	gradeState.Lock()
	gradeState.latestGrade = grade
	gradeState.hasLatestGrade = true
	gradeState.Unlock()
}

func LatestGradeInfoSnapshot() (protocol.StGradeInfo, bool) {
	gradeState.RLock()
	defer gradeState.RUnlock()
	return gradeState.latestGrade, gradeState.hasLatestGrade
}

func SetLatestExitInfosSnapshot(fsmID int32, exitInfos protocol.StExitInfos) {
	gradeState.Lock()
	gradeState.latestExitInfos = exitInfos
	gradeState.latestExitInfosFSM = fsmID
	gradeState.hasLatestExitInfos = true
	gradeState.Unlock()
}

func LatestExitInfosSnapshot() (int32, protocol.StExitInfos, bool) {
	gradeState.RLock()
	defer gradeState.RUnlock()
	return gradeState.latestExitInfosFSM, gradeState.latestExitInfos, gradeState.hasLatestExitInfos
}

func SetLatestMotorInfoSnapshot(fsmID int32, motor protocol.StMotorInfo) {
	gradeState.Lock()
	gradeState.latestMotor = motor
	gradeState.latestMotorFSM = fsmID
	gradeState.hasLatestMotor = true
	gradeState.Unlock()
}

func LatestMotorInfoSnapshot() (int32, protocol.StMotorInfo, bool) {
	gradeState.RLock()
	defer gradeState.RUnlock()
	return gradeState.latestMotorFSM, gradeState.latestMotor, gradeState.hasLatestMotor
}

func SetGradeInfoForDest(destID int32, grade protocol.StGradeInfo) {
	if destID == 0 {
		return
	}
	gradeState.Lock()
	gradeState.gradeInfoByDestID[destID] = grade
	gradeState.Unlock()
}

func LatestGradeInfoForDest(destID int32) (protocol.StGradeInfo, bool) {
	gradeState.RLock()
	grade, ok := gradeState.gradeInfoByDestID[destID]
	gradeState.RUnlock()
	return grade, ok
}
