---
title: Wash
---
Have you ever had to:

<details>
<summary>List all your AWS EC2 instances or Kubernetes pods?</summary>
<pre>aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --output text</pre>
<pre>kubectl get pods --all-namespaces`</pre>
</details>
<details>
<summary>Read/cat a GCP Compute instance's console output, or an AWS S3 object's content?</summary>
<pre>gcloud compute instances get-serial-port-output foo</pre>
<pre>aws s3api get-object content.txt --profile foo --bucket bar --key baz && cat content.txt && rm content.txt</pre>
</details>
<details>
<summary>Exec a command on a Kubernetes pod or GCP Compute Instance?</summary>
<pre>kubectl exec foo uname</pre>
<pre>gcloud compute ssh foo --command uname</pre>
</details>
<details>
<summary>Find all AWS EC2 instances with a particular tag, or Docker container with a specific label?</summary>
<pre>aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --filters Name=tag-key,Values=owner --output text</pre>
<pre>docker ps --filter “label=owner”</pre>
</details>

If not, then try clicking on the arrows to see the recommended way of doing those tasks. Does it bother you that each of those is a bespoke, cryptic incantation of various vendor-specific tools? It's a lot of commands you have to use, applications you need to install, and DSLs you have to learn just to do some pretty basic tasks. In Wash, these basic tasks are simple. You'll find that

<details>
<summary>Listing stuff is as easy as <code>ls</code></summary>
<pre>ls aws/foo/resources/ec2/instances</pre>
<pre>ls kubernetes/foo/bar/pods</pre>
</details>
<details>
<summary>Reading stuff is as easy as <code>cat</code>'ing a file</summary>
<pre>cat gcp/foo/compute/bar/console.out</pre>
<pre>cat aws/foo/resources/s3/bar/baz</pre>
</details>
<details>
<summary>Execing a command is as easy as <code>wexec</code></summary>
<pre>wexec kubernetes/foo/bar/pods/baz uname</pre>
<pre>wexec gcp/foo/compute/bar uname</pre>
</details>
<details>
<summary>Finding stuff is as easy as <code>find</code></summary>
<pre>find aws/foo -k '*ec2*instance' -meta '.tags[?].key' owner</pre>
<pre>find docker -k '*container' -meta '.labels.owner' -exists</pre>
</details>

And this is only scratching the surface of Wash's capabilities. Check out the screencast below

<script id="asciicast-MkbZKZZcmokHQ8z8jTv0PQs9F" src="https://asciinema.org/a/mX8Mwa75rr1bJePLi3OnIOkJK.js" async></script>

and the [tutorials](tutorials) to learn more.
