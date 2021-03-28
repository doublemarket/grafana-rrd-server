# Grafana RRD Server

A simple HTTP server that reads RRD files and responds to requests from Grafana with [Grafana Simple JSON Datasource plugin](https://grafana.net/plugins/grafana-simple-json-datasource).

[![CircleCI](https://img.shields.io/circleci/project/github/doublemarket/grafana-rrd-server.svg)](https://circleci.com/gh/doublemarket/grafana-rrd-server)
[![Coveralls](https://img.shields.io/coveralls/doublemarket/grafana-rrd-server.svg)](https://coveralls.io/github/doublemarket/grafana-rrd-server)
[![GitHub release](https://img.shields.io/github/release/doublemarket/grafana-rrd-server.svg)](https://github.com/doublemarket/grafana-rrd-server/releases)

This server supports all endpoints (urls) defined in the [Grafana Simple JSON Datasource plugin documentation](https://grafana.net/plugins/grafana-simple-json-datasource) but:

- You can use `*` as a wildcard in the `target` values (but not for `ds`) for the `/query` endpoint.

# Requirement

- librrd-dev (rrdtool)
- Go
- Grafana 3.0 and newer + Simple JSON Datasource plugin 1.0.0 and newer

# Usage

1. Install librrd-dev (rrdtool).

   On Ubuntu/Debian:

   ```
   sudo apt install librrd-dev
   ```

   On CentOS:

   ```
   sudo yum install rrdtool-devel
   ```

   On openSUSE
   ```
   sudo zypper in rrdtool-devel
   ```

   On Mac:

   ```
   brew install rrdtool
   ```

2. Get the package.

   ```
   go get github.com/doublemarket/grafana-rrd-server
   ```

   Otherwise, download [the latest release](https://github.com/doublemarket/grafana-rrd-server/releases/latest), gunzip it, and put the file in a directory included in `$PATH`:

   ```
   gunzip grafana-rrd-server_linux_amd64.gz
   ```

3. Run the server.

   ```
   grafana-rrd-server
   ```

   You can use the following options:

   - `-h` : Shows help messages.
   - `-p` : Specifies server port. (default: 9000)
   - `-i` : Specifies server listen address. (default: any)
   - `-r` : Specifies a directory path keeping RRD files. (default: "./sample/")
     - The server recursively searches RRD files under the directory and returns a list of them for the `/search` endpoint.
   - `-a` : Specifies the annotations file. It should be a CSV file which has a title line at the top like [the sample file](https://github.com/doublemarket/grafana-rrd-server/tree/master/sample/annotations.csv).
   - `-s` : Default graph step in second. (default: 10)
     - You can see the step for your RRD file using:
       ```
       $ rrdtool info [rrd file] | grep step
       step = 300
       ```

4. Optionally set up systemd unit:

```
useradd grafanarrd
cat > /etc/systemd/system/grafana-rrd-server.service <<EOF
[Unit]
Description=Grafana RRD Server
After=network.service

[Service]
User=grafanarrd
Group=grafanarrd
Restart=on-failure
Environment="LD_LIBRARY_PATH=/opt/rrdtool-1.6/lib"
ExecStart=/opt/grafana-rrd-server/grafana-rrd-server -p 9000 -r /path/to/rrds -s 300
RestartSec=10s

[Install]
WantedBy=default.target
EOF

systemctl daemon-reload
systemctl enable grafana-rrd-server
systemctl start grafana-rrd-server
```

5. Setup Grafana and Simple JSON Datastore plugin.

   See [Grafana documentation](http://docs.grafana.org/)

6. Create datasource.

# Docker (private repo) / Kubernetes how to:
1. build your image
```
$ docker build -t grafana-rrd-server .
```

2. run your server
```
$ docker run -p 9000:9000 -it grafana-rrd-server
```

3. list your image, get id
```
$ docker image ls
```

4. tag your release
```
$ docker tag <id> <repo>/grafana-rrd-server
```

5. push build to repo
```
$ docker push <repo>/grafana-rrd-server
```

6. pull from repo
```
$ docker pull <repo>/grafana-rrd-server
```

7. run local image from artifactory
```
$ docker run -p 9000:9000 -it <repo>/grafana-rrd-server
```

8. view sample data
```
  - $ open http://localhost:9000/search
  - $ curl -v http://localhost:9000/search
```

9. deploy grafana-rrd-server to k8s
$ TILLER_NAMESPACE=default helm install ./ --name pn-grafana-rrd-server --namespace=default -f ./values.yaml
$ TILLER_NAMESPACE=default helm upgrade pn-grafana-rrd-server ./ --reuse-values -f ./values.yaml

# Contributing

1. Install librrd-dev (rrdtool).

   See the Usage section.

2. Clone the repository.

3. Commit your code on a separate branch.

4. Create a pull request.

# License

MIT
