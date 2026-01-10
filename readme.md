一些有趣、简单、实用的go的库
1. ants：协程池
2. bigcache：本地缓存
3. cast：各种基础类型转换，省去断言的麻烦
4. errors：error的包裹，利用它就可以不必处处打印错误日志，将错误层层包裹上报统一打印，包括调用栈
5. jsonpath：在不知道具体的json结构的时候，可以用json path来尝试取出值；同时支持各种条件过滤
6. limiter：go官方的限流器，基于令牌桶的一种实现。在单实例的场景中简单实用，非分布式限流。
7. singleflight：并发控制，防击穿神器，比如保护DB击穿等等
8. ecdh: 基于椭圆曲线的密钥交换
9. echo：http的echo框架使用
10. metrics： go-metrics在Go性能指标度量中的应用
11. static: 将一些静态资源文件，编译成go文件
12. syncmap：官方syncMap库的注释版本
13. order-map: 基于插入顺序的有序map