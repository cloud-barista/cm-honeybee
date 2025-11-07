package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

func doGetInfraInfo(connID string) (*infra.Infra, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedInfraInfo, err := dao.SavedInfraInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the infra." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}
	var infraInfo infra.Infra
	err = json.Unmarshal([]byte(savedInfraInfo.InfraData), &infraInfo)
	if err != nil {
		errMsg := "Error occurred while parsing infra information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &infraInfo, nil
}

func doGetSoftwareInfo(connID string) (*software.Software, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedSoftwareInfo, err := dao.SavedSoftwareInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the software." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}
	var softwareInfo software.Software
	err = json.Unmarshal([]byte(savedSoftwareInfo.SoftwareData), &softwareInfo)
	if err != nil {
		errMsg := "Error occurred while parsing software information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &softwareInfo, nil
}

func doGetKubernetesInfo(connID string) (*kubernetes.Kubernetes, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedKubernetesInfo, err := dao.SavedKubernetesInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the kubernetes." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}
	var kubernetesInfo kubernetes.Kubernetes
	err = json.Unmarshal([]byte(savedKubernetesInfo.KubernetesData), &kubernetesInfo)
	if err != nil {
		errMsg := "Error occurred while parsing kubernetes information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &kubernetesInfo, nil
}

func doGetHelmInfo(connID string) (*kubernetes.Helm, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedHelmInfo, err := dao.SavedHelmInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the helm." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}
	var helmInfo kubernetes.Helm
	err = json.Unmarshal([]byte(savedHelmInfo.HelmData), &helmInfo)
	if err != nil {
		errMsg := "Error occurred while parsing helm information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &helmInfo, nil
}

// GetInfraInfo godoc
//
//	@ID				get-infra-info
//	@Summary		Get Infra Information
//	@Description	Get the infra information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	infra.Infra				"Successfully get information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the infra."
//	@Router			/source_group/{sgId}/connection_info/{connId}/infra [get]
func GetInfraInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	infraInfo, err := doGetInfraInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, *infraInfo, " ")
}

// GetInfraInfoSourceGroup godoc
//
//	@ID				get-infra-info-source-group
//	@Summary		Get Infra Information Source Group
//	@Description	Get the infra information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.InfraInfoList		"Successfully get information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the infra."
//	@Router			/source_group/{sgId}/infra [get]
func GetInfraInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var infraInfoList model.InfraInfoList

	for _, conn := range *list {
		infraInfo, _ := doGetInfraInfo(conn.ID)
		infraInfoList.Servers = append(infraInfoList.Servers, *infraInfo)
	}

	return c.JSONPretty(http.StatusOK, infraInfoList, " ")
}

// GetSoftwareInfo godoc
//
//	@ID				get-software-info
//	@Summary		Get Software Information
//	@Description	Get the software information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	software.Software		"Successfully get information of the software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the software."
//	@Router			/source_group/{sgId}/connection_info/{connId}/software [get]
func GetSoftwareInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	softwareInfo, err := doGetSoftwareInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}

// GetSoftwareInfoSourceGroup godoc
//
//	@ID				get-software-info-source-group
//	@Summary		Get Software Information Source Group
//	@Description	Get the software information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.SoftwareInfoList	"Successfully get information of the software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the software."
//	@Router			/source_group/{sgId}/software [get]
func GetSoftwareInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var softwareInfoList model.SoftwareInfoList

	for _, conn := range *list {
		softwareInfo, _ := doGetSoftwareInfo(conn.ID)
		softwareInfoList.Servers = append(softwareInfoList.Servers, *softwareInfo)
	}

	return c.JSONPretty(http.StatusOK, softwareInfoList, " ")
}

// GetKubernetesInfo godoc
//
//	@ID				get-kubernetes-info
//	@Summary		Get Kubernetes Information
//	@Description	Get the kubernetes information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	kubernetes.Kubernetes		"Successfully get information of the kubernetes."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the kubernetes."
//	@Router			/source_group/{sgId}/connection_info/{connId}/kubernetes [get]
func GetKubernetesInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	kubernetesInfo, err := doGetKubernetesInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, kubernetesInfo, " ")
}

// GetKubernetesInfoSourceGroup godoc
//
//	@ID				get-kubernetes-info-source-group
//	@Summary		Get Kubernetes Information Source Group
//	@Description	Get the kubernetes information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.KubernetesInfoList	"Successfully get information of the kubernetes."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the kubernetes."
//	@Router		/source_group/{sgId}/kubernetes [get]
func GetKubernetesInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var kubernetesInfoList model.KubernetesInfoList

	for _, conn := range *list {
		kubernetesInfo, _ := doGetKubernetesInfo(conn.ID)
		kubernetesInfoList.Servers = append(kubernetesInfoList.Servers, *kubernetesInfo)
	}

	return c.JSONPretty(http.StatusOK, kubernetesInfoList, " ")
}

// GetHelmInfo godoc
//
//	@ID				get-helm-info
//	@Summary		Get Helm Information
//	@Description	Get the helm information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	kubernetes.Helm		"Successfully get information of the kubernetes."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the kubernetes."
//	@Router			/source_group/{sgId}/connection_info/{connId}/helm [get]
func GetHelmInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	helmInfo, err := doGetHelmInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, helmInfo, " ")
}

// GetHelmInfoSourceGroup godoc
//
//	@ID				get-helm-info-source-group
//	@Summary		Get helm Information Source Group
//	@Description	Get the helm information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.HelmInfoList	"Successfully get information of the helm."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the helm."
//	@Router		/source_group/{sgId}/helm [get]
func GetHelmInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var helmInfoList model.HelmInfoList

	for _, conn := range *list {
		helmInfo, _ := doGetHelmInfo(conn.ID)
		helmInfoList.Servers = append(helmInfoList.Servers, *helmInfo)
	}

	return c.JSONPretty(http.StatusOK, helmInfoList, " ")
}

func doGetDataInfo(connID string) (*data.MinIOData, error) {
	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return nil, err
	}

	savedDataInfo, err := dao.SavedDataInfoGet(connectionInfo.ID)
	if err != nil {
		errMsg := "Failed to get information of the data." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}
	var dataInfo data.MinIOData
	err = json.Unmarshal([]byte(savedDataInfo.DataData), &dataInfo)
	if err != nil {
		errMsg := "Error occurred while parsing data information." +
			" (ConnectionID = " + connectionInfo.ID + ")"
		logger.Println(logger.ERROR, false, errMsg)
		return nil, errors.New(errMsg)
	}

	return &dataInfo, nil
}

// GetDataInfo godoc
//
//	@ID				get-data-info
//	@Summary		Get Data Information
//	@Description	Get the data information of the connection information.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	data.MinIOData		"Successfully get information of the data."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the data."
//	@Router			/source_group/{sgId}/connection_info/{connId}/data [get]
func GetDataInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	dataInfo, err := doGetDataInfo(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, dataInfo, " ")
}

// GetDataInfoSourceGroup godoc
//
//	@ID				get-data-info-source-group
//	@Summary		Get Data Information Source Group
//	@Description	Get the data information for all connections in the source group.
//	@Tags			[Get] Get source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	model.DataInfoList	"Successfully get information of the data."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the data."
//	@Router			/source_group/{sgId}/data [get]
func GetDataInfoSourceGroup(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var dataInfoList model.DataInfoList

	for _, conn := range *list {
		dataInfo, _ := doGetDataInfo(conn.ID)
		dataInfoList.MinIOData = append(dataInfoList.MinIOData, *dataInfo)
	}

	return c.JSONPretty(http.StatusOK, dataInfoList, " ")
}
