
在macos macbook air m1 里的ububtu虚拟机(2核4G)中的测试

firefox

一样是nginx + 自签名 作为tls 前置，speedtest-go 后置，

代理客户端和服务端同时放在 本机中运行，测试极限速度

nginx 最大上传缓存：51MB

## http, 本地回环

2325, 1546

## https，本地回环

```
1905，1883
1481，594
1873，1794
```

## 经过宿主机的clash的直连拷贝

（从虚拟机发送到宿主机，然后再被发回虚拟机）只是作为一种参考而已

970，664


## verysimple vless v0 tls

```
1702, 1329
1667，1309
1482，1362
1533，1122
```

## verysimple vless tls lazy encyrpt (splice)

```
1795，1432
1263，999
1868，1535
1836，1320
1640，957
1881，1452
1821，1521
```

## xray vless + xtls  direct

```
1655, 1291
1417, 1141
1680, 1263
```

## xray vless +xtls splice 

```
1737, 1280
1757, 1282
1772, 1297
```

# 总结

不错，在linux上的测试，verysimple已经是王者了，完胜xray！README里提供的测试也是类似的情况，不过那个虚拟机性能太差，我就单独在macbook上又安装了一个ubuntu虚拟机进行测试，发现效果是类似的。


另外在macos的ubuntu虚拟机中 还有一个毛病，就是它会使用宿主机macos的硬盘空间作为内存缓存；

也就是说，随着我不断的测速，我的虚拟机的硬盘占用量是越来越大的；（虽然关机后还可以reclaim，但是这确实值得注意）

发现，一般来说，verysimple的splice，在第一次测速时，效果都是特别好的，速度都不错，
而且每次开始第二次测速时，速度都会严重下滑；

我感觉速度下滑问题可能是内存泄漏问题没有解决好？因为遇到速度严重下滑的问题后，一般等待几秒再测试，似乎速度就又恢复正常了。

我认为这个问题是虚拟机的问题，可能和 占用宿主机硬盘空间 有一定关系。而且实际上测试本地回环的时候也出现过这种问题

还一个问题就是，下载速度基本完美，但是上传速度距离直连还有一定差距

还有就是，在macos上从来不会遇到服务器闪退的bug，但是在linux里就会有时遇到，但是也不频繁，测这么多次只遇到一次。


该ubuntu虚拟机上测速，客户端程序会呈现越来越慢的趋势，即第一次测速最快，然后后面的连续测速会越来越慢；停顿较长时间，至少三十秒后，能够恢复一部分速度。总之还是考虑这个虚拟机的问题。如果能真机测试效果最理想。但是谁又有linux真机呢，而且要求还带ui，好运行firefox。所以感觉，如果没有linux真机，可以考虑在软路由上测。虽然测到的不是极限速率，但是最起码不受虚拟机的性能以及底层的影响

如果该现象不仅在ubuntu虚拟机里发生的话，可能就是内存回收不够快导致的。产生了过多的buf放到了Pool里

另外，发现这个speedtest-go实际上也是多线程的。我把虚拟机调到4核心，速度又增加了，

4核心时，tls lazy encyrpt:

```
2346, 1771
2433, 1875

但是还是会不稳定地降速，下一次直接测到

758，514
再下一次
2130，1044
```

总之这种不稳情况在我的macos真机上是不存在的，所以肯定是虚拟机自己的问题了