package logstash

import (
	"bytes"
	"testing"
)

const exampleLogstashConfig = `
input {
  stdin {
    codec => "json_lines"
    type => "stdin"
    tags => [ "{{ ansible_hostname }}" ]

    tags => "hi"
    tags=>["a"]

  }
}
filter {
  mutate {
    gsub => [ "__REALTIME_TIMESTAMP", ".{3}$", "" ]
    rename => [ "MESSAGE", "message" ]
  }

    json {
      source => "message"
      add_tag => [ "json" ]
    }
  
  date {
    match => [ "__REALTIME_TIMESTAMP", "UNIX_MS"]
    timezone => "UTC"
  }
}
output {
  elasticsearch_http {
    host => "{{ elastic_host }}"
    index => "journald-%{+YYYY.MM.dd}"
    port => 9200
    document_id => "%{__CURSOR}"
  }
}
`

func TestParser_Parse(t *testing.T) {
	buf := bytes.NewBufferString(exampleLogstashConfig)
	p := NewParser(buf)
	c, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(c)
}
