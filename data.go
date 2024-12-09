package gdaldraw

import "fmt"

type Data struct {
	LatList     []float64 // 纬度需要从 -90~90
	LonList     []float64 // 经度需要从 -180~180
	Accuracy    float64   // 地图的精度
	ValueList   [][]float64
	OutFilePath string
}

func checkData(data *Data) error {
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
