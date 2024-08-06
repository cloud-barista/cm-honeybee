package controller

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// GetBenchmarkInfo godoc
//
//	@Summary		Get Benchmark Information
//	@Description	Get the benchmark information of the connection information.
//	@Tags			[Import] RunBenchmark
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connection info"
//	@Success		200	{object}	model.Benchmark			"Successfully get information of the benchmark."
//	@Success		200	{object}	model.SavedBenchmarkInfo		"Successfully get information of the benchmark."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the benchmark."
//	@Router			/honeybee/bench/{connId} [get]
func GetBenchmarkInfo(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedBenchmarkInfo, err := dao.SavedBenchmarkInfoGet(connectionInfo.ID)
	if err != nil {
		return common.ReturnErrorMsg(c, "Failed to get information of the benchmark.")
	}

	var benchmarkList []model.Benchmark
	err = json.Unmarshal([]byte(savedBenchmarkInfo.BenchmarkData), &benchmarkList)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while parsing software list.")
	}

	return c.JSONPretty(http.StatusOK, benchmarkList, " ")
}

// RunBenchmarkInfo godoc
//
//	@Summary		Run Benchmark Information
//	@Description	Run the benchmark information of the connection information.
//	@Tags			[Import] RunBenchmark
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connection info"
//	@Success		200	{object}	model.Benchmark			"Successfully get information of the benchmark."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the benchmark."
//	@Router			/honeybee/run/bench/{connId} [get]
func RunBenchmarkInfo(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	oldSavedBenchmarkInfo, _ := dao.SavedBenchmarkInfoGet(connectionInfo.ID)

	if oldSavedBenchmarkInfo == nil {
		savedBenchmarkInfo := new(model.SavedBenchmarkInfo)
		savedBenchmarkInfo.ConnectionID = connectionInfo.ID
		savedBenchmarkInfo.BenchmarkData = ""
		savedBenchmarkInfo.Status = "benchmarking"
		savedBenchmarkInfo.SavedTime = time.Now()
		savedBenchmarkInfo, err = dao.SavedBenchmarkInfoRegister(savedBenchmarkInfo)
		if err != nil {
			return common.ReturnInternalError(c, err, "Error occurred while getting infra information.")
		}
		oldSavedBenchmarkInfo = savedBenchmarkInfo
	}

	s := &ssh.SSH{
		Options: ssh.DefaultSSHOptions(),
	}

	data, err := s.RunBenchmark(*connectionInfo)
	if err != nil {
		oldSavedBenchmarkInfo.Status = "failed"
		_ = dao.SavedBenchmarkInfoUpdate(oldSavedBenchmarkInfo)
		return common.ReturnInternalError(c, err, "Error occurred while getting benchmark information.")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return common.ReturnErrorMsg(c, "Error occurred while converting benchmark data to JSON.")
	}

	oldSavedBenchmarkInfo.BenchmarkData = string(jsonData)
	oldSavedBenchmarkInfo.Status = "success"
	oldSavedBenchmarkInfo.SavedTime = time.Now()
	err = dao.SavedBenchmarkInfoUpdate(oldSavedBenchmarkInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, "Error occurred while saving the benchmark information.")
	}

	return c.JSONPretty(http.StatusOK, oldSavedBenchmarkInfo, " ")
}