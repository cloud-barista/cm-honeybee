// Getting ovs bridges for Linux

//go:build linux

package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/network"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"regexp"
	"time"

	"github.com/digitalocean/go-openvswitch/ovsdb"
	libovsdb "github.com/ovn-org/libovsdb/ovsdb"
)

func ovsSliceToGoNotation(val interface{}) (interface{}, error) {
	switch sl := val.(type) {
	case []interface{}:
		bsliced, err := json.Marshal(sl)
		if err != nil {
			return nil, err
		}
		switch sl[0] {
		case "uuid", "named-uuid":
			var uuid libovsdb.UUID
			err = json.Unmarshal(bsliced, &uuid)
			return ovsUUID{UUID: uuid.GoUUID}, err
		case "set":
			var oSet libovsdb.OvsSet
			err = json.Unmarshal(bsliced, &oSet)
			return ovsSet{Set: oSet.GoSet}, err
		case "map":
			var oMap libovsdb.OvsMap
			err = json.Unmarshal(bsliced, &oMap)
			return ovsMap{Map: oMap.GoMap}, err
		}
		return val, nil
	}
	return val, nil
}

func cleanInner(val []interface{}) ([]interface{}, error) {
	var newVal []interface{}

	for _, s := range val {
		var v []interface{}
		bs, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bs, &v)
		if err != nil {
			return nil, err
		}
		cleaned, err := ovsSliceToGoNotation(v)
		if err != nil {
			return nil, err
		}
		newVal = append(newVal, cleaned)
	}

	return newVal, nil
}

var validUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type ovsUUID struct {
	UUID string
}

func (u ovsUUID) validateUUID() error {
	if len(u.UUID) != 36 {
		return fmt.Errorf("uuid exceeds 36 characters")
	}

	if !validUUID.MatchString(u.UUID) {
		return fmt.Errorf("uuid does not match regexp")
	}

	return nil
}

func (u ovsUUID) MarshalJSON() ([]byte, error) {
	err := u.validateUUID()
	if err != nil {
		return nil, err
	}

	return json.Marshal(u.UUID)
}

type ovsSet struct {
	Set []interface{}
}

func (o ovsSet) MarshalJSON() ([]byte, error) {
	newVal, err := cleanInner(o.Set)
	if err != nil {
		return nil, err
	}

	return json.Marshal(newVal)
}

type ovsMap struct {
	Map map[interface{}]interface{}
}

func (o ovsMap) MarshalJSON() ([]byte, error) {
	if len(o.Map) > 0 {
		var innerMap []interface{}
		for key, val := range o.Map {
			var mapSeg []interface{}
			mapSeg = append(mapSeg, key)
			mapSeg = append(mapSeg, val)
			innerMap = append(innerMap, mapSeg)
		}
		newVal, err := cleanInner(innerMap)
		if err != nil {
			return nil, err
		}
		return json.Marshal(newVal)
	}
	return []byte("[]"), nil
}

var ovsDBSocketFile = "/var/run/openvswitch/db.sock"

func GetOVSInfo() ([]network.OVSBridge, error) {
	var ovsBridges []network.OVSBridge

	if !fileutil.IsExist(ovsDBSocketFile) {
		logger.Println(logger.DEBUG, true, "OVS: OVS database socket file not found.")

		return ovsBridges, nil
	}

	c, err := ovsdb.Dial("unix", "/var/run/openvswitch/db.sock")
	if err != nil {
		errMsg := fmt.Sprintf("failed to dial: %v", err)
		logger.Println(logger.ERROR, true, "OVS: "+errMsg)

		return ovsBridges, errors.New(errMsg)
	}
	defer func() {
		_ = c.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbs, err := c.ListDatabases(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list databases: %v", err)
		logger.Println(logger.ERROR, true, "OVS: "+errMsg)

		return ovsBridges, errors.New(errMsg)
	}

	var databaseFound = false
	var db string
	for _, db = range dbs {
		if db == "Open_vSwitch" {
			databaseFound = true
			break
		}
	}

	if !databaseFound {
		errMsg := "Open_vSwitch database not found"
		logger.Println(logger.ERROR, true, "OVS: "+errMsg)

		return ovsBridges, errors.New(errMsg)
	}

	rows, err := c.Transact(ctx, db, []ovsdb.TransactOp{
		ovsdb.Select{
			Table: "Bridge",
		},
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to perform transaction: %v", err)
		logger.Println(logger.ERROR, true, "OVS: "+errMsg)

		return ovsBridges, errors.New(errMsg)
	}

	type bridge map[string]interface{}

	var bridges []bridge

	for _, row := range rows {
		b := make(bridge)

		for key, val := range row {
			if key == "_uuid" {
				key = "uuid"
			} else if key == "_version" {
				key = "version"
			}

			val, err = ovsSliceToGoNotation(val)
			if err != nil {
				logger.Println(logger.ERROR, true, "OVS: "+err.Error())

				return ovsBridges, err
			}

			b[key] = val
		}
		bridges = append(bridges, b)
	}

	jsonBytes, err := json.Marshal(bridges)
	if err != nil {
		logger.Println(logger.ERROR, true, "OVS: "+err.Error())

		return ovsBridges, err
	}

	err = json.Unmarshal(jsonBytes, &ovsBridges)
	if err != nil {
		logger.Println(logger.ERROR, true, "OVS: "+err.Error())

		return ovsBridges, err
	}

	return ovsBridges, nil
}
