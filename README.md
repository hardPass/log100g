拙劣代码，勿被误导

代码背景，源自http://blog.sina.com.cn/s/blog_6cbb09f301015lch.html

	题目：
	每一个ip访问百度，其ip地址都会被记录到后台日志文件中，假设一天的访问日志有100G，
	求出一天中访问百度次数最多的ip地址，可以使用的内存大小是1G。

## 思路
首先是读一个大日志文件，处理过程中，根据IP4的第一个byte,把ip放到不同的小文件里，然后对每个小文件处理。如从大日志文件中读到ip:121.22.34.55，会把22.34.55(3个byte)写入121.part的小文件，然后后面会处理每个单独的小文件。100G的大日志文件，大概生成了255个小文件，总共1640 MB




* https://github.com/hardPass/log100g/blob/master/logFactory.go

	这个是生成大日志文件的一个代码，win7下生成100G文件用时34分钟


* https://github.com/hardPass/log100g/blob/master/logMaxIP_notfinished.go

	* 这个是 找出日志文件中ip的一个代码
	* 这个代码读日志用的是 file.read(b)，然后对内存 bytes.IndexByte...，
	* 对了名字叫做notfinished，不要太在意，代码捣腾玩的，没啥规范
	* 这个代码处理用时33分钟， 考虑io操作是大头，其实还可以考虑单线程顺序搞，速度会不会快点？
	
			LoopResovLine done!
			ipToDisk 1640 MB , all spend: 2019994 ms
			LoopToDisk done 1640 MB , all spend: 2019995 ms
			Done in 2082426 ms for 573312759 IPs!
			Max count  is: 31
			IP list is :
			 65.94.249.160




* 另外respo下面还有两个代码分别是用bufio读大日志的，设置了readerSize，貌似没什么明显效果

* 另外launchpad.net的mmap在windows下无法用，所以没搞，所以回头试一下java的randomAccessFile，速度会不会快些？