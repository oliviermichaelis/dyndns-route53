package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"io/ioutil"
	"log"
	"net"
	"time"
)

func getPublicIP(networkType string) ([]string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, networkType, "ns1.google.com:53")
		},
	}
	return r.LookupTXT(context.Background(), "o-o.myaddr.l.google.com")
}

func createParams(hostedZone *string, recordName *string, recordType string, recordValue string) *route53.ChangeResourceRecordSetsInput {
	return &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String(route53.ChangeActionUpsert),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: recordName,
						Type: aws.String(recordType),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(recordValue),
							},
						},
						TTL:           aws.Int64(60),
						Weight:        aws.Int64(1),
						SetIdentifier: aws.String(fmt.Sprintf("Update the %s record for %s", recordType, *recordName)),
					},
				},
			},
			Comment: aws.String(fmt.Sprintf("Update the %s record for %s", recordType, *recordName)),
		},
		HostedZoneId: hostedZone,
	}
}

func readSecret(filepath string, debug bool) (string, error) {
	b, err := ioutil.ReadFile(filepath)
	if debug {
		return "test", nil
	}

	return string(b), err
}

func main() {
	flagIPv4Enabled := flag.Bool("ipv4", false, "Enable IPv4 lookup")
	flagIPv6Enabled := flag.Bool("ipv6", false, "Enable IPv6 lookup")
	flagDebug := flag.Bool("debug", false, "Enable debug mode")
	flagAccessKeyID := flag.String("aws.accessKeyID", "", "AWS Access Key ID")
	flagSecretAccessKey := flag.String("aws.SecretAccessKey", "", "AWS Secret Access Key")
	flagRoute53HostedZoneName := flag.String("route53.hostedzone", "", "Name of HostedZone")
	flagRoute53AName := flag.String("route53.A.name", "", "Name of A record")
	flagRoute53AAAAName := flag.String("route53.AAAA.name", "", "Name of AAAA record")
	flag.Parse()

	if !*flagIPv4Enabled && !*flagIPv6Enabled {
		flag.Usage()
		log.Fatalf("Both ipv4 and ipv6 are false!")
	}

	if *flagIPv4Enabled && *flagRoute53AName == "" {
		flag.Usage()
		log.Fatalf("route53.A.name is missing")
	}

	if *flagIPv6Enabled && *flagRoute53AAAAName == "" {
		flag.Usage()
		log.Fatalf("route53.AAAA.name is missing")
	}

	accessKey, err := readSecret(*flagAccessKeyID, *flagDebug)
	if err != nil {
		log.Fatal(err)
	}

	secretKey, err := readSecret(*flagSecretAccessKey, *flagDebug)
	if err != nil {
		log.Fatal(err)
	}

	ips := make(map[string][]string)
	if *flagIPv4Enabled {
		ip, err := getPublicIP("udp4")
		if err != nil {
			fmt.Println(err)
		}
		ips["ipv4"] = ip
	}

	if *flagIPv6Enabled {
		ip, err := getPublicIP("udp6")
		if err != nil {
			fmt.Println(err)
		}
		ips["ipv6"] = ip
	}

	//s := session.Must(session.NewSessionWithOptions(session.Options{
	//	SharedConfigState: session.SharedConfigEnable,
	//}))

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})

	r := route53.New(sess)
	if *flagIPv4Enabled {
		params := createParams(flagRoute53HostedZoneName, flagRoute53AName, route53.RRTypeA, ips["ipv4"][0])
		resp, err := r.ChangeResourceRecordSets(params)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(resp)
	}

	// This is kind of broken, since it doesnt get the IPv6 of the gateway but rather of the host this binary is running on
	if *flagIPv6Enabled {
		params := createParams(flagRoute53HostedZoneName, flagRoute53AName, route53.RRTypeAaaa, ips["ipv6"][0])
		resp, err := r.ChangeResourceRecordSets(params)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(resp)
	}
}
