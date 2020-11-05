## 使用go_plugin

1. Plguin 需要有自己的 main package
2. 编译的时候，使用 go build -buildmode=plugin file.go 来编译
3. 使用 plugin.Open(path string) 来打开.so文件，同一插件只能打开一次，重复打开会报错
4. 使用 plugin.LookUp(name string) 来获取插件中对外暴露的方法或者类型

### http_trigger服务
执行顺序：
1. 请求localhost:9100/upload 上传一段代码，模版代码如下：
```golang
package main
import "net/http"

func CustomHandler(w http.ResponseWriter, r *http.Request) {
    // your logic code
}
```
在CustomHandler编写需要的代码逻辑

2. 上传完成后请求localhost:9100/specialize，将上传的代码编译为动态库，并获取对应函数名的句柄
3. 请求localhost:9100/，即可运行上传的代码。

### 题外话
开源的fission在加载用户自定的函数实际上就是使用了plugin技术。感兴趣的可以去看一下fission
