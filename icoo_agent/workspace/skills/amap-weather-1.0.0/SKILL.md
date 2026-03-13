---
name: AMap Weather
slug: amap-weather
version: 1.0.0
description: Query weather information using AMap (高德地图) Weather API. Supports real-time weather and 4-day forecasts for any city in China.
metadata: { "openclaw": { "emoji": "🌤️", "requires": { "bins": ["python3"], "env": ["AMAP_WEATHER_KEY"] }, "primaryEnv": "AMAP_WEATHER_KEY" } }
---

# AMap Weather 🌤️

Query weather information for any city in China using AMap Weather API.

## Features

- **Real-time weather**: Current temperature, humidity, wind, weather condition
- **4-day forecast**: Day/night weather, temperature range, wind direction
- **City lookup**: By adcode (行政区划代码) or city name (支持中文城市名)
- **Chinese support**: All weather descriptions in Chinese
- **Full city database**: 3000+ cities from AMap administrative data

## Usage

```bash
python3 scripts/weather.py '<JSON>'
```

## Request Parameters

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| city | str | yes | - | City adcode (e.g., "110101") or city name (e.g., "北京", "上海") |
| extensions | str | no | base | "base" for real-time, "all" for forecast |
| key | str | no | env | AMap API key (uses AMAP_WEATHER_KEY env var if not provided) |

## Examples

```bash
# Real-time weather by city name
python3 scripts/weather.py '{"city": "北京"}'
python3 scripts/weather.py '{"city": "上海"}'
python3 scripts/weather.py '{"city": "深圳"}'

# Real-time weather by adcode
python3 scripts/weather.py '{"city": "110101"}'

# 4-day forecast
python3 scripts/weather.py '{"city": "广州", "extensions": "all"}'

# With explicit API key
python3 scripts/weather.py '{"city": "杭州", "key": "your_api_key"}'
```

## Output Format

### Real-time Weather (base)

```json
{
  "status": "success",
  "data": {
    "province": "北京",
    "city": "东城区",
    "adcode": "110101",
    "weather": "晴",
    "temperature": "9",
    "winddirection": "东南",
    "windpower": "≤3",
    "humidity": "41",
    "reporttime": "2026-03-12 11:03:40"
  }
}
```

### Forecast (all)

```json
{
  "status": "success",
  "data": {
    "province": "上海",
    "city": "上海市",
    "adcode": "310000",
    "reporttime": "2026-03-12 11:02:01",
    "forecasts": [
      {
        "date": "2026-03-12",
        "week": "4",
        "dayweather": "晴",
        "nightweather": "晴",
        "daytemp": "15",
        "nighttemp": "6",
        "daywind": "北",
        "nightwind": "北",
        "daypower": "1-3",
        "nightpower": "1-3"
      }
      // ... 4 days total
    ]
  }
}
```

## Error Handling

Returns error object on failure:

```json
{
  "status": "error",
  "error": "City not found",
  "code": "INVALID_CITY"
}
```

## Supported Cities

Supports 3000+ cities across China. Common examples:

| City | Adcode |
|------|--------|
| 北京 | 110000 |
| 上海 | 310000 |
| 广州 | 440100 |
| 深圳 | 440300 |
| 杭州 | 330100 |
| 成都 | 510100 |

For full city lookup, see `region_by_name.py` in the skill directory.

## Data Files

The skill includes comprehensive administrative division data:

- `region_by_name.py` - City name to adcode mapping (3200+ entries)
- `region_by_adcode.py` - Adcode to city info mapping
- `region_list.json` - Full data as JSON array
- `region_nested.json` - Hierarchical province-city-district structure

## API Limits

- Free tier: 5,000 calls/day
- Rate limit: 50 calls/second

## Current Status

Fully functional.