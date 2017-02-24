# Nosee
A nosey, agentless, easy monitoring tool over SSH.

**Heavy WIP!**

What is it?
-----------

It's an answer when you found usual monitoring systems too heavy and complex.

Nosee uses SSH protocol to execute scripts on monitored systems, checking
for whatever you want. The result is checked and Nosee will ring an alert
of your choice if anything is wrong.

In short : SSH, no agent, simple configuration, usual scripting.

Currently, Nosee requires bash on the monitored machine. It was tested successfully
tested with Linux (of course) but using Cygwin for Windows hosts too.


How do you build it?
--------------------

If you have Go installed:

	go get github.com/Xfennec/nosee

You will then be able to launch the binary located in you Go "bin" directory.


How do you use it?
------------------

![Nosee general configuration structure](https://raw.github.com/Xfennec/nosee/master/doc/images/img_general.png)

Step by step.
