apache_logpipe
==============

About this project
------------------

**Don't laught about this code :-) This project is only a training project that I use to build up a luttle bit knowledge in the programming language GO.**
Therefore it is a wise idea not to use this in production :-)
A lot of things are currently probably terribly awkward for GO professionals and implemented in a non-idiomatic way.

I have not spent very much time in reading books or documentations about this programming language, 
but I plan to do so in the next weeks. Then my code will probably improve ;-)

**Feedback is very welcome :-)**

Features
--------

* Rotate logfile similar to cronolog and logrotate2
* Analyze accesslogs
  * calculate performance statistics
  * group performance statisics by regular expressions
  * handle static content seperatly 
* send statistics to zabbix


SCRATCHPAD:
------------------
```
go get -d ./...
go get -u -v -f all
```
https://golang.org/doc/effective_go.html
https://play.golang.org/
https://medium.com/@_orcaman/most-imported-golang-packages-some-insights-fb12915a07

https://medium.com/rate-engineering/go-test-your-code-an-introduction-to-effective-testing-in-go-6e4f66f2c259
https://golang.org/dl/
https://www.alexedwards.net/blog/an-overview-of-go-tooling


TODO:
----

- Refactor to more object oriented code
- Write unittests
- Add zabbix statistics submission
  (https://github.com/adubkov/go-zabbix)
- Implement ansynchronous statistics calculation
- Implement ansynchronous statistics submission to zabbix
- Add a signal handler which closes logfile ad output statistics
- Write documentation
- Maintain a "current" link for the logfile
- Add configuration file
- Understand go dependency management

DONE:
-----

- Add a logging framework
  https://godoc.org/github.com/golang/glog
  https://gobyexample.com/command-line-flags
- Parse loglines
- Calculate statistics
- Add getopts parser
- Write logfile


Installation an usage
---------------------

* Compile and build project
  ```
  make release
  scp apache_logpipe <webserver>:/usr/local/bin
  ```
* Add this directive to your /etc/apache2.conf or apache vhost config
  ```
  LogFormat "%h %v:%p %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\" %D" 
  CustomLog "|/usr/local/bin -zabbix_server zabbix.mydomain.org -output_logfile '/var/log/apache2/access.log.%Y-%m-%d" vhost_combined_canonical
  ```
* Restart Apache
  ```
  /etc/init.d/apache2 reload
  ```
* Add zabbix template to zabbix and assign it to the host


