拙劣代码，勿被误导

https://github.com/hardPass/log100g/blob/master/logFactory.go
这个是生成大日志文件的一个代码


https://github.com/hardPass/log100g/blob/master/logMaxIP_notfinished.go
这个是 找出日志文件中ip的一个代码
这个代码读日志用的是bytes.Index...， file.read(b)


另外respo下面还有两个代码分别是用bufio读大日志的，设置了readerSize，貌似没什么明显效果

mmap在windows下无法用，所以没搞