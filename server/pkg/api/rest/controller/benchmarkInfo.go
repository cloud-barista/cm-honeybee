package controller

import (
//	"encoding/json"
	"net/http"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/lib/ssh"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"github.com/jollaman999/utils/logger"
)

// GetBenchmarkInfo godoc
//
//	@Summary		Get Benchmark Information
//	@Description	Get the benchmark information of the connection information.
//	@Tags			[Import] BenchmarkInfo
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

	logger.Println(logger.INFO, true, "savedBenchmarkInfo : ", savedBenchmarkInfo)

	return c.JSONPretty(http.StatusOK, savedBenchmarkInfo, " ")
}

// RunBenchmarkInfo godoc
//
//	@Summary		Run Benchmark Information
//	@Description	Run the benchmark information of the connection information.
//	@Tags			[Import] BenchmarkInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connection info"
//	@Success		200	{object}	model.Benchmark			"Successfully get information of the benchmark."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the benchmark."
//	@Router			/honeybee/bench/{connId}/run [get]
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
		savedBenchmarkInfo.Benchmark = ""
		savedBenchmarkInfo.Status = "benchmarking"
		savedBenchmarkInfo.SavedTime = time.Now()
		savedBenchmarkInfo, err = dao.SavedBenchmarkInfoRegister(savedBenchmarkInfo)
		if err != nil {
			return common.ReturnInternalError(c, err, "Error occurred while getting benchmark information.")
		}
		oldSavedBenchmarkInfo = savedBenchmarkInfo
	}

	s := &ssh.SSH{
		Options: ssh.DefaultSSHOptions(),
	}


	oldSavedBenchmarkInfo.Status = "benchmarking"
	_ = dao.SavedBenchmarkInfoUpdate(oldSavedBenchmarkInfo)
	
	typeStr := c.QueryParam("types")

	go func(typeStr string, benchmarkInfo *model.SavedBenchmarkInfo) {
		benchmarkData, _ := s.RunBenchmark(*connectionInfo, typeStr)
		if err != nil {
			logger.Println(logger.DEBUG, true, err.Error())
		}

		benchmarkInfo.Status = "success"
		benchmarkInfo.Benchmark = benchmarkData

		err = dao.SavedBenchmarkInfoUpdate(benchmarkInfo)
		if err != nil {
			logger.Println(logger.ERROR, true, "err is : ", err)
		}

	}(typeStr, oldSavedBenchmarkInfo)

	return c.JSONPretty(http.StatusOK, oldSavedBenchmarkInfo, " ")
}

// StopBenchmarkInfo godoc
//
//	@Summary		Stop Benchmark
//	@Description	Stop the benchmark
//	@Tags			[Import] BenchmarkInfo
//	@Accept			json
//	@Produce		json
//	@Param			connId path string true "ID of the connection info"
//	@Success		200	{object}	model.SimpleMsg				"Benchmark Stopped."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to stop of the benchmark."
//	@Router			/honeybee/bench/{connId}/stop [get]
func StopBenchmarkInfo(c echo.Context) error {
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
		return common.ReturnInternalError(c, err, "Error occurred while getting benchmark information.")
	}

	s := &ssh.SSH{
		Options: ssh.DefaultSSHOptions(),
	}

	err = s.StopBenchmark(*connectionInfo)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}

	oldSavedBenchmarkInfo.Status = "stopped"
	oldSavedBenchmarkInfo.SavedTime = time.Now()
	err = dao.SavedBenchmarkInfoUpdate(oldSavedBenchmarkInfo)
	if err != nil {
		return common.ReturnErrorMsg(c, "Error occurred while saving the benchmark information.")
	}

	return c.JSONPretty(http.StatusOK, oldSavedBenchmarkInfo, " ")
}