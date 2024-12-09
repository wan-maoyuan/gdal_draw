package gdaldraw

import (
	"fmt"
)

type Data struct {
	LatList     []float64 // 纬度需要从 北纬90 到 南纬90
	LonList     []float64 // 经度需要从 西经180 到 东经180
	Accuracy    float64   // 地图的精度
	ValueList   [][]float64
	OutFilePath string
}

func (data *Data) check() error {
	if len(data.LatList) == 0 {
		return fmt.Errorf("lat slice is empty")
	}

	if len(data.LonList) == 0 {
		return fmt.Errorf("lon slice is empty")
	}

	if len(data.ValueList) == 0 {
		return fmt.Errorf("value slice is empty")
	}

	if len(data.LatList) != len(data.ValueList) {
		return fmt.Errorf("value first slice length not equal lat slice length")
	}

	if len(data.LonList) != len(data.ValueList[0]) {
		return fmt.Errorf("value second slice length not equal lon slice length")
	}

	if data.Accuracy <= 0 {
		return fmt.Errorf("lat lon accuracy is invalid")
	}

	if data.OutFilePath == "" {
		return fmt.Errorf("out file path is empty")
	}

	return nil
}

type DoubleData struct {
	LatList     []float64 // 纬度需要从 北纬90 到 南纬90
	LonList     []float64 // 经度需要从 西经180 到 东经180
	Accuracy    float64   // 地图的精度
	Value1List  [][]float64
	Value2List  [][]float64
	OutFilePath string
}

func (data *DoubleData) check() error {
	if len(data.LatList) == 0 {
		return fmt.Errorf("lat slice is empty")
	}

	if len(data.LonList) == 0 {
		return fmt.Errorf("lon slice is empty")
	}

	if len(data.Value1List) == 0 {
		return fmt.Errorf("value1 slice is empty")
	}

	if len(data.Value2List) == 0 {
		return fmt.Errorf("value2 slice is empty")
	}

	if len(data.LatList) != len(data.Value1List) {
		return fmt.Errorf("value1 first slice length not equal lat slice length")
	}

	if len(data.LatList) != len(data.Value2List) {
		return fmt.Errorf("value2 first slice length not equal lat slice length")
	}

	if len(data.LonList) != len(data.Value1List[0]) {
		return fmt.Errorf("value1 second slice length not equal lon slice length")
	}

	if len(data.LonList) != len(data.Value2List[0]) {
		return fmt.Errorf("value2 second slice length not equal lon slice length")
	}

	if data.Accuracy <= 0 {
		return fmt.Errorf("lat lon accuracy is invalid")
	}

	if data.OutFilePath == "" {
		return fmt.Errorf("out file path is empty")
	}

	return nil
}

type IrregularData struct {
	LatList     []float64 // 纬度范围： 90 到 -90
	LonList     []float64 // 经度范围：-180 到 180
	ValueList   []float64
	Accuracy    float64 // 地图的精度
	OutFilePath string
}

func (data *IrregularData) check() error {
	if len(data.LatList) == 0 {
		return fmt.Errorf("lat slice is empty")
	}

	if len(data.LonList) == 0 {
		return fmt.Errorf("lon slice is empty")
	}

	if len(data.ValueList) == 0 {
		return fmt.Errorf("value slice is empty")
	}

	if len(data.LatList) != len(data.ValueList) {
		return fmt.Errorf("value slice length not equal lat slice length")
	}

	if data.Accuracy <= 0 {
		return fmt.Errorf("lat lon accuracy is invalid")
	}

	if data.OutFilePath == "" {
		return fmt.Errorf("out file path is empty")
	}

	return nil
}
