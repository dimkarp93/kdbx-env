package keepass

import "testing"

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile><Root><Group><Name>Root</Name>
  <Entry><String><Key>Title</Key><Value>GITHUB_TOKEN</Value></String><String><Key>Password</Key><Value>ghp_top</Value></String></Entry>
  <Group><Name>web</Name>
    <Entry><String><Key>Title</Key><Value>API_KEY</Value></String><String><Key>Password</Key><Value>web_secret</Value></String></Entry>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>web_dup</Value></String></Entry>
  </Group>
  <Group><Name>db</Name>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>db_dup</Value></String></Entry>
  </Group>
</Group></Root></KeePassFile>`

func TestParseSecrets(t *testing.T) {
	entries, err := ParseSecrets(sampleXML)
	if err != nil {
		t.Fatal(err)
	}

	v, err := LookupSecret(entries, "GITHUB_TOKEN")
	if err != nil || v != "ghp_top" {
		t.Errorf("GITHUB_TOKEN: got %q, err %v", v, err)
	}

	v, err = LookupSecret(entries, "API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("API_KEY: got %q, err %v", v, err)
	}

	v, err = LookupSecret(entries, "web/API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("web/API_KEY by path: got %q, err %v", v, err)
	}

	if _, err := LookupSecret(entries, "DUP"); err == nil {
		t.Error("expected ambiguity error for DUP")
	}

	v, err = LookupSecret(entries, "db/DUP")
	if err != nil || v != "db_dup" {
		t.Errorf("db/DUP by path: got %q, err %v", v, err)
	}

	if _, err := LookupSecret(entries, "MISSING"); err == nil {
		t.Error("expected not-found error for MISSING")
	}
}
