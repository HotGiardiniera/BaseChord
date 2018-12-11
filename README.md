# Chord Implementation and Enhancements
<!-- ![NYU logo](https://github.com/HotGiardiniera/BaseChord/blob/master/Images/nyu_logo.jpg =50x) -->

<b>Authors:</b> Chris Moirano and Yair Schiff \
<b>Course:</b> NYU Fall 2018 Distributed Systems (CSCI-GA 3033-002) Project\
<b>Instructor:</b> Aurojit Panda

This project implements the [Chord algorithm](https://pdos.csail.mit.edu/papers/ton:chord/paper-ton.pdf) for P2P file systems (with the proper changes noted in [Zave, 2012](https://arxiv.org/pdf/1502.06461.pdf) that make the algorithm correct).

In addition to the baseline algorithm implementation, two enhancements, detailed below, are also implemented.

## Usage
### Client

The client allows interaction through a ring node.
Three simple actions are allowed: <b>get</b>, <b>store</b>, <b>delete</b>

Client takes three flags:
1. `-endpoint=<node IP>`. Connects to a ring node endpoint.
2. `-call=<get store delete>`. Applies an action at a node.
3. `-file=<filename>`. The file we want to apply an action to.


Example usage of client below
```
vagrant@stretch[client]$ ./client -endpoint=stretch:3002 -call=store -file=test
2018/11/30 21:36:58 Connecting to  stretch:3002
2018/11/30 21:36:58 Storing file: "test"
2018/11/30 21:36:58 stretch:3002 response: File "test" successfully stored/deleted!

vagrant@stretch[client]$ ./client -endpoint=stretch:3002 -call=get -file=test
2018/11/30 21:37:04 Connecting to  stretch:3002
2018/11/30 21:37:04 Getting file: "test"
2018/11/30 21:37:04 stretch:3002 response: File "test" found! Data: test

vagrant@stretch[client]$ ./client -endpoint=stretch:3002 -call=delete -file=test
2018/11/30 21:37:10 Connecting to  stretch:3002
2018/11/30 21:37:10 Deleting file: "test"
2018/11/30 21:37:10 stretch:3002 response: File "test" successfully stored/deleted!

vagrant@stretch[client]$ ./client -endpoint=stretch:3002 -call=get -file=test
2018/11/30 21:37:12 Connecting to  stretch:3002
2018/11/30 21:37:12 Getting file: "test"
2018/11/30 21:37:12 stretch:3002 response: File "test" not found!
```


### Metrics
Metrics are gatherd as JSON objects. Each request is it's own JSON object. The Go struct associated with the returned JSON object can be found at $PROJECTROOT/chord/metric.go. Metric objects are created for every request that comes to a chord node. The metric object is then placed into a channel. At a certain interval (interval located at $PROJECTROOT/chord/utils.go `MetricsTimeout`) the metric buffer is then drained and each metric object is appended to a JSON file specific to each node. The JSON file is located the path specified in $PROJECTROOT/chord/metric.go as `METRICSFILE`. By default this will be `/tmp/node<ID>.JSON`.

TODO
1. Better JSON file output (single rows right now, make a list)
