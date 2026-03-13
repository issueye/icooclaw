#!/usr/bin/env python3
"""
AMap Weather API Client
高德地图天气查询工具

Usage:
    python3 scripts/weather.py '{"city": "110101"}'
    python3 scripts/weather.py '{"city": "北京", "extensions": "all"}'
"""

import sys
import os
import json
import requests
from typing import Optional, Dict, Any
from pathlib import Path

# API configuration
AMAP_WEATHER_URL = "https://restapi.amap.com/v3/weather/weatherInfo"
DEFAULT_KEY = os.environ.get("AMAP_WEATHER_KEY", "6bbf77fed36fa3c1ea9ba263d6de8dba")

# 数据文件路径
SKILL_DIR = Path(__file__).parent
REGION_BY_NAME_FILE = SKILL_DIR / "region_by_name.py"
REGION_BY_ADCODE_FILE = SKILL_DIR / "region_by_adcode.py"

# 常用城市adcode映射（快速查找）
CITY_ADCODE_MAP = {
    # 直辖市
    "北京": "110000", "北京市": "110000",
    "天津": "120000", "天津市": "120000",
    "上海": "310000", "上海市": "310000",
    "重庆": "500000", "重庆市": "500000",
    # 省会城市
    "石家庄": "130100", "石家庄市": "130100",
    "太原": "140100", "太原市": "140100",
    "沈阳": "210100", "沈阳市": "210100",
    "长春": "220100", "长春市": "220100",
    "哈尔滨": "230100", "哈尔滨市": "230100",
    "南京": "320100", "南京市": "320100",
    "杭州": "330100", "杭州市": "330100",
    "合肥": "340100", "合肥市": "340100",
    "福州": "350100", "福州市": "350100",
    "南昌": "360100", "南昌市": "360100",
    "济南": "370100", "济南市": "370100",
    "郑州": "410100", "郑州市": "410100",
    "武汉": "420100", "武汉市": "420100",
    "长沙": "430100", "长沙市": "430100",
    "广州": "440100", "广州市": "440100",
    "深圳": "440300", "深圳市": "440300",
    "南宁": "450100", "南宁市": "450100",
    "海口": "460100", "海口市": "460100",
    "成都": "510100", "成都市": "510100",
    "贵阳": "520100", "贵阳市": "520100",
    "昆明": "530100", "昆明市": "530100",
    "拉萨": "540100", "拉萨市": "540100",
    "西安": "610100", "西安市": "610100",
    "兰州": "620100", "兰州市": "620100",
    "西宁": "630100", "西宁市": "630100",
    "银川": "640100", "银川市": "640100",
    "乌鲁木齐": "650100", "乌鲁木齐市": "650100",
    # 特别行政区
    "香港": "810000",
    "澳门": "820000",
    # 其他热门城市
    "苏州": "320500", "苏州市": "320500",
    "无锡": "320200", "无锡市": "320200",
    "宁波": "330200", "宁波市": "330200",
    "厦门": "350200", "厦门市": "350200",
    "青岛": "370200", "青岛市": "370200",
    "大连": "210200", "大连市": "210200",
    "东莞": "441900", "东莞市": "441900",
    "佛山": "440600", "佛山市": "440600",
    "珠海": "440400", "珠海市": "440400",
    "三亚": "460200", "三亚市": "460200",
}

# 缓存完整城市映射
_full_region_map = None


def _load_full_region_map() -> Dict[str, str]:
    """加载完整的城市名称到adcode映射"""
    global _full_region_map
    if _full_region_map is not None:
        return _full_region_map
    
    # 从内置映射开始
    _full_region_map = dict(CITY_ADCODE_MAP)
    
    # 尝试加载完整数据
    try:
        if REGION_BY_NAME_FILE.exists():
            # 读取Python文件并提取字典
            content = REGION_BY_NAME_FILE.read_text(encoding='utf-8')
            # 简单解析：找到字典定义
            import ast
            # 找到 REGION_BY_NAME = 后面的字典
            start = content.find('{')
            end = content.rfind('}') + 1
            if start != -1 and end > start:
                dict_str = content[start:end]
                data = ast.literal_eval(dict_str)
                for name, info in data.items():
                    if isinstance(info, dict) and 'adcode' in info:
                        _full_region_map[name] = str(info['adcode'])
    except Exception:
        pass
    
    return _full_region_map


def resolve_city(city: str) -> str:
    """
    Resolve city name to adcode.
    If city is already an adcode (6 digits), return as-is.
    Otherwise, look up in the city map.
    """
    # If already an adcode (6 digits), return as-is
    if city.isdigit() and len(city) == 6:
        return city
    
    # 先查快速映射
    if city in CITY_ADCODE_MAP:
        return CITY_ADCODE_MAP[city]
    
    # 加载完整映射
    full_map = _load_full_region_map()
    if city in full_map:
        return full_map[city]
    
    # 尝试模糊匹配
    city_lower = city.lower()
    for name, adcode in full_map.items():
        if city_lower in name.lower() or name.lower() in city_lower:
            return adcode
    
    # Return original if not found (API might still work)
    return city


def query_weather(
    city: str,
    extensions: str = "base",
    key: Optional[str] = None
) -> Dict[str, Any]:
    """
    Query weather information from AMap API.

    Args:
        city: City adcode or city name
        extensions: "base" for real-time, "all" for forecast
        key: AMap API key (uses default if not provided)

    Returns:
        Dict with weather data or error info
    """
    api_key = key or DEFAULT_KEY
    
    # 解析城市名称为adcode
    adcode = resolve_city(city)

    params = {
        "city": adcode,
        "key": api_key,
        "extensions": extensions
    }

    try:
        response = requests.get(AMAP_WEATHER_URL, params=params, timeout=10)
        response.raise_for_status()
        data = response.json()

        # Check API response status
        if data.get("status") != "1":
            return {
                "status": "error",
                "error": data.get("info", "Unknown error"),
                "code": data.get("infocode", "UNKNOWN")
            }

        # Process based on extension type
        if extensions == "all":
            # Forecast mode
            forecasts = data.get("forecasts", [])
            if not forecasts:
                return {
                    "status": "error",
                    "error": "No forecast data available",
                    "code": "NO_DATA"
                }
            
            forecast_data = forecasts[0]
            return {
                "status": "success",
                "data": {
                    "province": forecast_data.get("province"),
                    "city": forecast_data.get("city"),
                    "adcode": forecast_data.get("adcode"),
                    "reporttime": forecast_data.get("reporttime"),
                    "forecasts": [
                        {
                            "date": cast.get("date"),
                            "week": cast.get("week"),
                            "dayweather": cast.get("dayweather"),
                            "nightweather": cast.get("nightweather"),
                            "daytemp": cast.get("daytemp"),
                            "nighttemp": cast.get("nighttemp"),
                            "daywind": cast.get("daywind"),
                            "nightwind": cast.get("nightwind"),
                            "daypower": cast.get("daypower"),
                            "nightpower": cast.get("nightpower")
                        }
                        for cast in forecast_data.get("casts", [])
                    ]
                }
            }
        else:
            # Real-time mode
            lives = data.get("lives", [])
            if not lives:
                return {
                    "status": "error",
                    "error": "No weather data available for this city",
                    "code": "NO_DATA"
                }
            
            live_data = lives[0]
            return {
                "status": "success",
                "data": {
                    "province": live_data.get("province"),
                    "city": live_data.get("city"),
                    "adcode": live_data.get("adcode"),
                    "weather": live_data.get("weather"),
                    "temperature": live_data.get("temperature"),
                    "winddirection": live_data.get("winddirection"),
                    "windpower": live_data.get("windpower"),
                    "humidity": live_data.get("humidity"),
                    "reporttime": live_data.get("reporttime")
                }
            }
            
    except requests.exceptions.Timeout:
        return {
            "status": "error",
            "error": "Request timeout",
            "code": "TIMEOUT"
        }
    except requests.exceptions.RequestException as e:
        return {
            "status": "error",
            "error": str(e),
            "code": "NETWORK_ERROR"
        }
    except json.JSONDecodeError:
        return {
            "status": "error",
            "error": "Invalid JSON response",
            "code": "PARSE_ERROR"
        }


def format_output(result: Dict[str, Any], extensions: str = "base") -> str:
    """Format weather result as human-readable text."""
    if result.get("status") == "error":
        return f"❌ 错误: {result.get('error')} (代码: {result.get('code')})"
    
    data = result.get("data", {})
    
    if extensions == "all":
        # Forecast format
        lines = [
            f"📍 {data.get('province')} {data.get('city')} ({data.get('adcode')})",
            f"📅 更新时间: {data.get('reporttime')}",
            ""
        ]
        
        for forecast in data.get("forecasts", []):
            week_map = {"1": "周一", "2": "周二", "3": "周三", "4": "周四", 
                       "5": "周五", "6": "周六", "7": "周日"}
            week = week_map.get(forecast.get("week", ""), forecast.get("week", ""))
            
            lines.append(f"📆 {forecast.get('date')} {week}")
            lines.append(f"   ☀️ 白天: {forecast.get('dayweather')} {forecast.get('daytemp')}°C")
            lines.append(f"   🌙 夜间: {forecast.get('nightweather')} {forecast.get('nighttemp')}°C")
            lines.append(f"   💨 风向: {forecast.get('daywind')}/{forecast.get('nightwind')} {forecast.get('daypower')}/{forecast.get('nightpower')}级")
            lines.append("")
        
        return "\n".join(lines)
    else:
        # Real-time format
        return "\n".join([
            f"📍 {data.get('province')} {data.get('city')} ({data.get('adcode')})",
            f"🌤️ 天气: {data.get('weather')}",
            f"🌡️ 温度: {data.get('temperature')}°C",
            f"💧 湿度: {data.get('humidity')}%",
            f"💨 风向: {data.get('winddirection')} {data.get('windpower')}级",
            f"📅 更新: {data.get('reporttime')}"
        ])


def main():
    """Main entry point."""
    # Set stdin encoding for Windows
    if sys.platform == 'win32':
        sys.stdin.reconfigure(encoding='utf-8')
    
    if len(sys.argv) < 2:
        # Try reading from stdin
        if not sys.stdin.isatty():
            try:
                input_data = sys.stdin.read()
                params = json.loads(input_data)
            except json.JSONDecodeError as e:
                print(json.dumps({
                    "status": "error",
                    "error": f"Invalid JSON from stdin: {e}",
                    "code": "INVALID_INPUT"
                }, ensure_ascii=False))
                sys.exit(1)
        else:
            print("Usage: python3 scripts/weather.py '<JSON>'")
            print('Example: python3 scripts/weather.py \'{"city": "110101"}\'')
            sys.exit(1)
    else:
        try:
            # Handle potential encoding issues on Windows
            arg = sys.argv[1]
            if isinstance(arg, bytes):
                arg = arg.decode('utf-8')
            params = json.loads(arg)
        except json.JSONDecodeError as e:
            print(json.dumps({
                "status": "error",
                "error": f"Invalid JSON: {e}",
                "code": "INVALID_INPUT"
            }, ensure_ascii=False))
            sys.exit(1)
    
    city = params.get("city")
    if not city:
        print(json.dumps({
            "status": "error",
            "error": "Missing required parameter: city",
            "code": "MISSING_CITY"
        }, ensure_ascii=False))
        sys.exit(1)
    
    extensions = params.get("extensions", "base")
    key = params.get("key")
    
    result = query_weather(city, extensions, key)
    
    # Output JSON
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()