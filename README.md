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
$ export GIN_MODE=release
$ go run main.go &
```
The first line redirects incoming connections for port 443 to port 8080, where the website
is running. Port 443 is the where HTTPS requests come in to the machine, and they need
to be redirected to the actual server running in port 8080. The server runs in port 8080
by default.
