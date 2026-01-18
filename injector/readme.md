# 基于go的注入器，在全局空间中注入变量

如果是单例模式，每个结构体指针注入对象在内存中只能有一个实例，
我们设置两次到命名映射中以确保这一点（第一次按名称，然后按结构体指针类型）
1.if singleton, every struct ptr inject object can only have one
instance in memory, we set twice in named to
ensure that(first by name, then by struct ptr type)

2.TODO every struct inject object(not a ptr) should be handled like:
create new struct on every inject

	`inject:""` by type,must be a pointer to a struct
	`inject:"devService"`

3.singleton(default false)

	`singleton:"true"`
	`singleton:"false"`

4.cannil(default false)

	`cannil:"true"`
	`cannil:"false"`