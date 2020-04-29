module github.com/ecordell/cop

go 1.13

require (
	github.com/dghubble/oauth1 v0.6.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/juju/go4 v0.0.0-20160222163258-40d72ab9641a // indirect
	github.com/juju/persistent-cookiejar v0.0.0-20171026135701-d5e5a8405ef9
	github.com/manifoldco/promptui v0.7.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
	golang.org/x/net v0.0.0-20190912160710-24e19bdeb0f2
	gopkg.in/andygrunwald/go-jira.v1 v1.8.0
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/retry.v1 v1.0.3 // indirect
	k8s.io/test-infra v0.0.0-20200107123819-bffa19577291 // indirect
)

replace github.com/juju/persistent-cookiejar v0.0.0-20171026135701-d5e5a8405ef9 => github.com/orirawlings/persistent-cookiejar v0.0.0-20181119224032-99f11603c3cf
