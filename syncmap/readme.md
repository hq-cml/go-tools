原则：
Locked函数都是表示在锁的保护下
新加入到 map 中的 key 会被放到 dirty 中，到了一定程度会，将dirty=>read，提升命中率


https://zhuanlan.zhihu.com/p/44585993
https://blog.csdn.net/u010230794/article/details/82143179