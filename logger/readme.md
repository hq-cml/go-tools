### 一个简单好用的日志库，底层封装了log4go
增加了自动行数获取__LINE____，自动文件名__FILE__
1. %M	消息内容。即你调用 log.Info(“hello”) 时传入的 “hello”。
2. %L	日志级别。例如：INFO, DEBUG, ERROR 等。
3. %D	长格式日期。格式为 2006/01/02。
4. %d	短格式日期。格式为 01/02/06。
5. %T	长格式时间。格式为 15:04:05 MST（时:分:秒 时区）。
6. %t	短格式时间。格式为 15:04（时:分）。

Demo:

    <logging>
        <!-- 输出到控制台 -->
        <filter enabled="true">
            <tag>stdout</tag>
            <type>console</type>
            <level>DEBUG</level>
            <property name="format">[%D %T] [%L] %M</property>
        </filter>
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
`
`