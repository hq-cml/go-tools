# Go Logger

一个基于 log4go 封装的日志库，自动添加文件名和行号信息。

## 特性

- ✓ 自动记录调用位置（文件名:行号）
- ✓ 五级日志：Debug、Info、Warn、Error、Critical
- ✓ 支持配置文件（XML）
- ✓ 日志滚动（按大小/天数/行数）
- ✓ 多输出：控制台 + 多文件

## 安装

```bash
go get github.com/hq-cml/go-tools/logger
```

## 快速开始

```go
package main

import (
    log "github.com/hq-cml/go-tools/logger"
)

func main() {
    // 加载配置文件
    log.LoadConfiguration("log.xml")
    defer log.Close()  // 确保日志写入
    
    // 使用日志
    log.Info("应用启动")
    log.Error("发生错误: %v", err)
}
```

## 配置示例 (log.xml)

```xml
<logging>
    <!-- 输出到控制台 -->
    <filter enabled="true">
        <tag>stdout</tag>
        <type>console</type>
        <level>DEBUG</level>
        <property name="format">[%D %T] [%L] %M</property>
    </filter>
    
    <!-- 应用日志文件 -->
    <filter enabled="true">
        <!-- 定义filter的key，不能重复 -->
        <tag>app-file</tag>
        <type>file</type>
        <level>INFO</level>
        <!-- 输出到文件，并配置滚动规则 -->
        <property name="filename">/tmp/applog/app.log</property>
        <!-- 在这里定义格式 -->
        <property name="format">[%D %T] [%L] %M</property>
        <!-- 启用滚动 -->
        <property name="rotate">true</property>     
        <property name="maxsize">100M</property>    <!-- 单个文件最大100MB -->
        <property name="maxlines">100000</property> <!-- 单个文件最多10万行 -->
        <property name="daily">true</property>      <!-- 按天分割 -->
    </filter>
    
    <!-- 错误日志文件（单独记录WARNING及以上级别） -->
    <filter enabled="true">
        <!-- 定义filter的key，不能重复 -->
        <tag>err-file</tag>
        <type>file</type>
        <level>WARNING</level>
        <!-- 输出到文件，并配置滚动规则 -->
        <property name="filename">/tmp/applog/app_err.log</property>
        <!-- 在这里定义格式 -->
        <property name="format">[%D %T] [%L] %M</property>
        <!-- 启用滚动 -->
        <property name="rotate">true</property>
        <property name="maxsize">100M</property>    <!-- 单个文件最大100MB -->
        <property name="maxlines">100000</property> <!-- 单个文件最多10万行 -->
        <property name="daily">true</property>      <!-- 按天分割 -->
    </filter>
</logging>
```

## 格式占位符

| 占位符 | 说明 |
|--------|------|
| %M | 消息内容。即你调用 log.Info("hello") 时传入的 "hello"。 |
| %L | 日志级别。例如：INFO, DEBUG, ERROR 等。 |
| %D | 长格式日期。格式为 2006/01/02。 |
| %d | 短格式日期。格式为 01/02/06。 |
| %T | 长格式时间。格式为 15:04:05 MST（时:分:秒 时区）。 |
| %t | 短格式时间。格式为 15:04（时:分）。 |

## ⚠️ 重要提示

log4go 默认使用异步写入，如果主程序结束过快，日志可能来不及写入。务必使用 `defer log.Close()` 或在 main 函数结束时调用 `log.Close()`。
