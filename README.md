This website exists because I wanted to buy my name's domain name. Just hacking around on it.
The CNAME file tells Github to redirect traffic from spwg.github.io to spencerwgreene.com.

The code that runs the website is in the http directory. Main.go runs a gin-gonic web server
written in Go that serves static files from the http/site directory. To run the web server
in development, clone the repository, open a command line prompt, and run
```console
$ git clone https://github.com/spwg/spencerwgreene.com.git
$ cd spencerwgreene.com/http
$ go run main.go &
```
The web server runs on port 8080 by default. So, go to localhost:8080 in a browser to see
the web site running.

To deploy the website, ssh into the GCP instance, run tmux, clone it if it's not already cloned, 
change directory into the git repository, and run
```console
$ sudo iptables -A PREROUTING -t nat -p tcp --dport 443 -j REDIRECT --to-ports 8080
$ sudo iptables -A PREROUTING -t nat -p tcp --dport 80 -j REDIRECT --to-ports 8081
$ export GIN_MODE=release
$ go run main.go &
```
The first line redirects incoming connections for port 443 to port 8080, where the website
is running with HTTPS certificates. Port 443 is the where HTTPS requests come in to the machine.
The second line redirects incoming connections for port 80 to port 8081, where the HTTP version
of the site is running. Port 80 is where HTTP requests come in to the machine. You don't need
to run these commands more than once per server. The ip tables should look like this:
```console
$ sudo iptables --table nat --list
Chain PREROUTING (policy ACCEPT)
target     prot opt source               destination
REDIRECT   tcp  --  anywhere             anywhere             tcp dpt:https redir ports 8080
REDIRECT   tcp  --  anywhere             anywhere             tcp dpt:http redir ports 8081

Chain INPUT (policy ACCEPT)
target     prot opt source               destination

Chain OUTPUT (policy ACCEPT)
target     prot opt source               destination

Chain POSTROUTING (policy ACCEPT)
target     prot opt source               destination
```
