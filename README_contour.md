# 等值线生成功能说明

## 功能概述

本模块提供了等值线生成功能，可以将不规则分布的气象数据点转换为等值线，并输出为GeoJSON格式。

## 主要功能

### DrawContourLines 函数

```go
func DrawContourLines(data *ContourLinesData) error
```

该函数接收 `ContourLinesData` 结构体作为输入，生成等值线并保存为GeoJSON文件。

## 数据结构

### ContourLinesData

```go
type ContourLinesData struct {
    LatList     []float64 // 纬度范围： 90 到 -90
    LonList     []float64 // 经度范围：-180 到 180
    ValueList   []float64 // 所有点的值
    Accuracy    float64   // 地图的精度（经纬度步长）
    Step        float64   // 等值线间距
    OutFilePath string    // 输出GeoJSON文件路径
}
```

## 使用示例

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/wan-maoyuan/gdal_draw"
)

func main() {
    // 创建数据
    data := &gdaldraw.ContourLinesData{
        LatList:     []float64{90, 88, 86, ...}, // 纬度列表
        LonList:     []float64{-180, -178, -176, ...}, // 经度列表
        ValueList:   []float64{25.5, 26.1, 24.8, ...}, // 对应的值
        Accuracy:    2.0,  // 2度精度
        Step:        5.0,  // 等值线间隔5度
        OutFilePath: "contour_lines.geojson",
    }
    
    // 生成等值线
    err := gdaldraw.DrawContourLines(data)
    if err != nil {
        fmt.Printf("生成等值线失败: %v\n", err)
        return
    }
    
    fmt.Println("等值线生成成功！")
}
```

### 运行示例

```bash
cd example
go run main.go
```

这将生成以下文件：
- `temp_3857.png` - 3857投影的温度图
- `temp_4326.png` - 4326投影的温度图
- `contour_lines.geojson` - 等值线GeoJSON文件

## 算法说明

### 1. 数据插值

- 将不规则分布的点数据插值到规则网格
- 使用反距离加权插值算法填充空白点
- 支持NaN值的处理

### 2. 等值线生成

- 使用Marching Squares算法生成等值线
- 自动计算数据的最小值和最大值
- 根据指定的步长生成等值线值

### 3. GeoJSON输出

- 将等值线转换为标准的GeoJSON格式
- 每条等值线包含其对应的值
- 支持LineString几何类型

## 输出格式

生成的GeoJSON文件包含以下结构：

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": {
        "type": "LineString",
        "coordinates": [[lon1, lat1], [lon2, lat2], ...]
      },
      "properties": {
        "value": 25.0
      }
    }
  ]
}
```

## 参数说明

- **Accuracy**: 经纬度网格的精度，建议值：0.25-2.0度
- **Step**: 等值线的间隔，建议值：1.0-10.0度
- **LatList/LonList**: 输入点的经纬度坐标
- **ValueList**: 对应的气象数据值（如温度、气压等）

## 注意事项

1. 输入数据的经纬度范围必须正确（纬度：90到-90，经度：-180到180）
2. 三个列表的长度必须相同
3. 建议使用较小的精度值以获得更平滑的等值线
4. 等值线间隔应根据数据的分布范围合理设置

## 性能优化

- 对于大数据集，建议先进行数据采样
- 可以通过调整搜索半径来平衡精度和性能
- 考虑使用并行处理来加速插值过程 