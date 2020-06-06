apache_logpipe
==============

About this project
------------------

This tool can be used to gather statistics on your apache webserver and send them to the zabbix monitoring system.
Apache provides the possibility to use a subprocess which reads from stdin as logging destination.
This tool collects the loglines and writes the to a logfile asynchronously comparable to the [cronolog](https://github.com/fordmason/cronolog) utility.
It also asynchronously calculates delivery statistics for virtual hosts and send zabbix discoveries and data to zabbix for that hosts.
This was my first go training project ;-)


Features
--------

* Rotate logfile similar to cronolog and logrotate2
* Analyze accesslogs
  * calculate performance statistics
  * group performance statistics by regular expressions
  * handle static content separately 
* send statistics to zabbix


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
  CustomLog "|/usr/local/bin --zabbix_server zabbix.mydomain.org --output_logfile '/var/log/apache2/access.log.%Y-%m-%d" vhost_combined_canonical
  ```
* Restart Apache
  ```
  /etc/init.d/apache2 reload
  ```
* Add zabbix template to zabbix and assign it to the host


TODOs and Ideas:
----------------

- write zipped logfiles (https://gist.github.com/mchirico/6147687)
- Add a simple webserver which provides configuring intefaces/statistics
- Understand go dependency management
- Make the code more modular/structured 
- Write documentation

DONE:
-----

- Add configuration file
- Refactor to more object oriented and understandable code
- Output statistics
- Make logstream implementation threadsafe
  (this is not neccessary in general, but interesting from academic view)
- Maintain a "current" link for the logfile
- Add a signal handler which closes logfile and output statistics
- Implement ansynchronous statistics calculation
- Implement ansynchronous statistics submission to zabbix
- Implement zabbix discovery
- Add a logging framework
  https://godoc.org/github.com/golang/glog
  https://gobyexample.com/command-line-flags
- Parse loglines
- Calculate statistics
- Add option parser
- Write logfile
- Write unittests
  - https://github.com/golang/mock
  - https://blog.codecentric.de/2019/07/gomock-vs-testify/
