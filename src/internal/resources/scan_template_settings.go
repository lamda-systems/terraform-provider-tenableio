package resources

// The typed settings of a user-defined scan template.
//
// Each entry below is one setting Tenable.io documents for the request body of
// POST /scans ("Create scan"), which the policy endpoints point at for the
// meaning of every setting a policy stores. Kinds, allowed values and ranges
// are taken from that schema; descriptions are its first sentence. A policy
// stores the same settings as a scan apart from targets, so the scan-only ones
// are left out: name and description (top-level attributes here), folder,
// scanner, targets, schedule, notifications and permissions. So are the
// settings that name an uploaded file, which need the file upload endpoint.
//
// When a template's own form definition (GET /editor/policy/templates/{uuid})
// becomes available as a catalogue, entries can be added here without
// changing anything else: the schema, the request, the read-back and the echo
// check are all driven from this table.
//
// Where the documented enum and its own prose disagree, choices hold the union.
// A validator that rejected a value the docs call valid would block a working
// configuration; one that admits an invalid value only lets the API reject it,
// and the echo check reports that with both values on screen.

// settingBlock is the nested block a setting is configured in. The blocks
// follow the sections of the template editor in the Tenable.io UI.
type settingBlock string

const (
	blockDiscovery  settingBlock = "discovery"
	blockAssessment settingBlock = "assessment"
	blockReport     settingBlock = "report"
	blockAdvanced   settingBlock = "advanced"
	blockAgent      settingBlock = "agent"
)

// settingKind is how a setting is typed in Terraform and written on the wire.
type settingKind int

const (
	// kindYesNo is a bool in Terraform and "yes" or "no" on the wire.
	kindYesNo settingKind = iota
	// kindChoice is a string restricted to choices.
	kindChoice
	// kindNumber is an integer in Terraform and a decimal string on the wire.
	kindNumber
	// kindInteger is an integer in Terraform and on the wire.
	kindInteger
	// kindText is a free-form string.
	kindText
)

// templateFamily is the kind of template a setting applies to. Tenable.io
// documents the settings of network templates and of agent templates
// separately, and discards a setting the template does not define.
type templateFamily int

const (
	familyAny templateFamily = iota
	familyNetwork
	familyAgent
)

type scanSetting struct {
	block   settingBlock
	key     string
	kind    settingKind
	family  templateFamily
	choices []string
	// min and max bound a numeric setting when both are non-zero.
	min, max    int64
	description string
}

// scanSettings is generated from the Create scan reference
// (https://developer.tenable.com/reference/scans-create), plus
// allow_post_scan_editing from the policy details reference.
var scanSettings = []scanSetting{
	// -- discovery -------------------------------------------------------------
	{block: blockDiscovery, key: "wmi_netstat_scanner", kind: kindYesNo, family: familyAny,
		description: "Specifies whether the scanner uses 'netstat' to determine open ports while performing a WMI-based scan."},
	{block: blockDiscovery, key: "ssh_netstat_scanner", kind: kindYesNo, family: familyAny,
		description: "Specifies whether the scanner uses 'netstat' to identify open ports via an SSH connection."},
	{block: blockDiscovery, key: "discovery_mode", kind: kindChoice, family: familyAny, choices: []string{"Port scan (common ports)", "Port scan (all ports)", "Custom"},
		description: "Specifies the preconfigured discovery mode for the scan. When set to a value other than 'Custom', the system applies predefined host discovery, ping, and port scanning settings."},
	{block: blockDiscovery, key: "ping_the_remote_host", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner pings remote hosts on multiple ports to determine their availability before beginning the scan."},
	{block: blockDiscovery, key: "fast_network_discovery", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner uses an accelerated discovery process."},
	{block: blockDiscovery, key: "arp_ping", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner pings a host using its hardware address via the Address Resolution Protocol (ARP)."},
	{block: blockDiscovery, key: "tcp_ping", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner pings a host using TCP."},
	{block: blockDiscovery, key: "tcp_ping_dest_ports", kind: kindText, family: familyNetwork,
		description: "Specifies the destination ports checked via TCP ping. You can specify 'built-in', a single port, or a comma-separated list of ports."},
	{block: blockDiscovery, key: "icmp_ping", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner pings a host using the Internet Control Message Protocol (ICMP)."},
	{block: blockDiscovery, key: "icmp_unreach_means_host_down", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the system considers a targeted host dead when the scanner receives an ICMP unreachable message from a gateway."},
	{block: blockDiscovery, key: "icmp_ping_retries", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of attempts the system makes to retry pinging a remote host."},
	{block: blockDiscovery, key: "udp_ping", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner pings a host using the User Datagram Protocol (UDP)."},
	{block: blockDiscovery, key: "scan_network_printers", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner scans network printers. When enabled, you can choose to surface the printer OS only or perform a full network scan."},
	{block: blockDiscovery, key: "scan_netware_hosts", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to scan Novell Netware hosts."},
	{block: blockDiscovery, key: "scan_ot_devices_with_discovery", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner uses protocol-specific queries (via OT Discovery) to identify live Operational Technology (OT) devices and retrieve granular attributes such as make, model, and firmware."},
	{block: blockDiscovery, key: "scan_ot_devices", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs a full scan of Operational Technology (OT) devices."},
	{block: blockDiscovery, key: "wol_wait_time", kind: kindNumber, family: familyNetwork,
		description: "Specifies the amount of time, in minutes, the system waits for hosts to start after sending Wake-on-LAN (WOL) magic packets before performing the scan."},
	{block: blockDiscovery, key: "unscanned_closed", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner considers unscanned ports as closed."},
	{block: blockDiscovery, key: "portscan_range", kind: kindText, family: familyNetwork,
		description: "Specifies the range of ports to be scanned. The 'default' keyword instructs the scanner to scan approximately 4,790 commonly used ports."},
	{block: blockDiscovery, key: "snmp_scanner", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner uses SNMP to enumerate open ports and retrieve detailed host information."},
	{block: blockDiscovery, key: "only_portscan_if_enum_failed", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs a traditional port scan only if local port enumerators (such as SSH, WMI, or SNMP) fail to retrieve the port list."},
	{block: blockDiscovery, key: "verify_open_ports", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs additional verification to confirm that a port identified as open by a port enumerator is actually accessible."},
	{block: blockDiscovery, key: "tcp_scanner", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs a full TCP connect scan to identify open ports."},
	{block: blockDiscovery, key: "tcp_firewall_detection", kind: kindChoice, family: familyNetwork, choices: []string{"Do not detect RST rate limitation (soft)", "Ignore closed ports (aggressive)", "Disabled (softer)", "Automatic (normal)"},
		description: "Specifies the method used to detect firewalls during a TCP scan."},
	{block: blockDiscovery, key: "syn_scanner", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs a TCP SYN (half-open) scan to identify open ports."},
	{block: blockDiscovery, key: "syn_firewall_detection", kind: kindChoice, family: familyNetwork, choices: []string{"Do not detect RST rate limitation (soft)", "Ignore closed ports (aggressive)", "Disabled (softer)", "Automatic (normal)"},
		description: "Specifies the method used to detect firewalls during a SYN scan."},
	{block: blockDiscovery, key: "udp_scanner", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner performs a UDP scan to identify open ports."},
	{block: blockDiscovery, key: "svc_detection_on_all_ports", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner attempts to identify the service running on every open port identified during the scan."},
	{block: blockDiscovery, key: "detect_ssl", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner attempts to detect SSL/TLS encryption on open ports."},
	{block: blockDiscovery, key: "ssl_prob_ports", kind: kindChoice, family: familyNetwork, choices: []string{"All ports", "Known SSL ports", "None", "Known ports"},
		description: "Specifies which ports the scanner probes for SSL/TLS encryption."},
	{block: blockDiscovery, key: "dtls_prob_ports", kind: kindChoice, family: familyNetwork, choices: []string{"None", "Known DTLS ports", "All ports", "Known ports"},
		description: "Specifies which ports the scanner probes for DTLS encryption."},
	{block: blockDiscovery, key: "cert_expiry_warning_days", kind: kindNumber, family: familyNetwork,
		description: "Specifies the number of days before an SSL/TLS certificate expires that the system generates a warning finding."},
	{block: blockDiscovery, key: "enumerate_all_ciphers", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner attempts to enumerate all supported SSL/TLS ciphers for each detected service."},
	{block: blockDiscovery, key: "check_crl", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner checks the Certificate Revocation List (CRL) to determine if a detected SSL/TLS certificate has been revoked."},
	{block: blockDiscovery, key: "collect_identity_data_from_ad", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to collect identity data from Active Directory."},

	// -- assessment ------------------------------------------------------------
	{block: blockAssessment, key: "override_accuracy", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to override the normal accuracy settings for plugins."},
	{block: blockAssessment, key: "report_paranoia", kind: kindChoice, family: familyAny, choices: []string{"Normal", "Avoid false alarms", "Paranoid (more false alarms)"},
		description: "Specifies the paranoia level for reporting potential vulnerabilities."},
	{block: blockAssessment, key: "potential_vulns_components", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to assess component installs for potential vulnerabilities."},
	{block: blockAssessment, key: "thorough_tests", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to perform thorough tests. When enabled, plugins use more in-depth detection methods."},
	{block: blockAssessment, key: "adsi_query", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to use ADSI queries for Windows user enumeration."},
	{block: blockAssessment, key: "wmi_query", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to use WMI queries for Windows user enumeration."},
	{block: blockAssessment, key: "samr_enumeration", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to use SAMR for Windows user enumeration."},
	{block: blockAssessment, key: "rid_brute_forcing", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to use RID brute forcing for Windows user enumeration."},
	{block: blockAssessment, key: "request_windows_domain_info", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to request information about the SMB domain."},
	{block: blockAssessment, key: "enum_domain_users_start_uid", kind: kindNumber, family: familyAny,
		description: "Specifies the starting UID for domain user enumeration."},
	{block: blockAssessment, key: "enum_domain_users_end_uid", kind: kindNumber, family: familyAny,
		description: "Specifies the ending UID for domain user enumeration."},
	{block: blockAssessment, key: "enum_local_users_start_uid", kind: kindNumber, family: familyAny,
		description: "Specifies the starting UID for local user enumeration."},
	{block: blockAssessment, key: "enum_local_users_end_uid", kind: kindNumber, family: familyAny,
		description: "Specifies the ending UID for local user enumeration."},
	{block: blockAssessment, key: "assessment_mode", kind: kindChoice, family: familyAny, choices: []string{"Default", "Scan for known web vulnerabilities", "Scan for all web vulnerabilities (quick)", "Scan for all web vulnerabilities (complex)", "Custom"},
		description: "Specifies the preconfigured assessment mode for the scan. When set to a value other than 'Custom', the system applies predefined vulnerability assessment settings."},
	{block: blockAssessment, key: "collect_crypto_inventory", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to collect cryptographic asset inventory data during the scan."},
	{block: blockAssessment, key: "av_grace_period", kind: kindNumber, family: familyAny, min: 0, max: 7,
		description: "Specifies the antivirus definition grace period in days (0-7)."},
	{block: blockAssessment, key: "smtp_domain", kind: kindText, family: familyNetwork,
		description: "Specifies the third-party SMTP domain. This domain must be outside the range of the host being scanned or the site performing the scan."},
	{block: blockAssessment, key: "smtp_from", kind: kindText, family: familyNetwork,
		description: "Specifies the SMTP sender (FROM) address."},
	{block: blockAssessment, key: "smtp_to", kind: kindText, family: familyNetwork,
		description: "Specifies the SMTP recipient (TO) address."},
	{block: blockAssessment, key: "provided_creds_only", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to only use credentials provided by the user and not attempt to log in with default accounts or passwords."},
	{block: blockAssessment, key: "test_default_oracle_accounts", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to test default Oracle database accounts."},
	{block: blockAssessment, key: "modbus_start_reg", kind: kindNumber, family: familyNetwork,
		description: "Specifies the starting register for Modbus/TCP coil access scanning."},
	{block: blockAssessment, key: "modbus_end_reg", kind: kindNumber, family: familyNetwork,
		description: "Specifies the ending register for Modbus/TCP coil access scanning."},
	{block: blockAssessment, key: "start_cotp_tsap", kind: kindNumber, family: familyNetwork,
		description: "Specifies the starting COTP TSAP for ICCP/COTP TSAP addressing weakness scanning."},
	{block: blockAssessment, key: "stop_cotp_tsap", kind: kindNumber, family: familyNetwork,
		description: "Specifies the stopping COTP TSAP for ICCP/COTP TSAP addressing weakness scanning."},
	{block: blockAssessment, key: "scan_webapps", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to enable web application scanning. When enabled, the scanner crawls web applications and tests for vulnerabilities."},
	{block: blockAssessment, key: "webcrawler_start_page", kind: kindText, family: familyNetwork,
		description: "Specifies the start page for the web crawler. Use a colon delimiter to specify multiple start pages (for example, '/:/php4:/base')."},
	{block: blockAssessment, key: "webcrawl_exclude_regex", kind: kindText, family: familyNetwork,
		description: "Specifies a regular expression pattern for URLs the crawler should exclude."},
	{block: blockAssessment, key: "webcrawler_max_pages", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of pages to crawl."},
	{block: blockAssessment, key: "webcrawl_max_depth", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum depth of the crawl."},
	{block: blockAssessment, key: "webcrawl_follow_dynamic_pages", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the web crawler should follow dynamically generated pages."},
	{block: blockAssessment, key: "generic_webapp_tests", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to enable generic web application tests (such as SQL injection and XSS)."},
	{block: blockAssessment, key: "abort_generic_webapp_if_login_fails", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to abort web application tests if the HTTP login fails."},
	{block: blockAssessment, key: "try_all_http_methods", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to test all HTTP methods (POST in addition to default GET) for web form testing."},
	{block: blockAssessment, key: "http_param_pollution", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to attempt HTTP parameter pollution attacks."},
	{block: blockAssessment, key: "test_embedded_web_servers", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to test embedded web servers. Embedded web servers are often static and have no customizable CGI."},
	{block: blockAssessment, key: "combo_arg_values", kind: kindChoice, family: familyNetwork, choices: []string{"some pairs", "all pairs (slower but efficient)", "some combinations", "all combinations (extremely slow)"},
		description: "Specifies how many parameter combinations to test per form when testing more than one parameter at a time."},
	{block: blockAssessment, key: "stop_at_first_flaw", kind: kindChoice, family: familyNetwork, choices: []string{"per port (quicker)", "per parameter (slow)", "look for all flaws (slower)"},
		description: "Specifies when to stop testing for web application flaws on a given page."},
	{block: blockAssessment, key: "url_for_rfi", kind: kindText, family: familyNetwork,
		description: "Specifies the URL for Remote File Inclusion (RFI) testing."},
	{block: blockAssessment, key: "generic_webapp_tests_max_time", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum run time in minutes for each individual generic web attack type (such as XSS, SQL injection)."},
	{block: blockAssessment, key: "user_agent_string", kind: kindText, family: familyNetwork,
		description: "Specifies a custom user-agent string to use for HTTP requests during the scan."},
	{block: blockAssessment, key: "custom_http_header_enabled", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to add a custom header to all HTTP requests."},
	{block: blockAssessment, key: "custom_http_header_name", kind: kindText, family: familyNetwork,
		description: "Specifies the name of the custom HTTP header."},
	{block: blockAssessment, key: "custom_http_header_value", kind: kindText, family: familyNetwork,
		description: "Specifies the value of the custom HTTP header."},
	{block: blockAssessment, key: "oracle_database_use_detected_sids", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to use detected SIDs for Oracle database scanning."},
	{block: blockAssessment, key: "scan_malware", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to enable malware scanning on target hosts."},
	{block: blockAssessment, key: "enable_file_scanning", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to enable file system scanning for malware."},
	{block: blockAssessment, key: "enable_filescan_systemroot", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the Windows %SystemRoot% directory."},
	{block: blockAssessment, key: "enable_filescan_programfiles", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the Windows Program Files directory."},
	{block: blockAssessment, key: "enable_filescan_programfilesx86", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the Windows Program Files (x86) directory."},
	{block: blockAssessment, key: "enable_filescan_programdata", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the Windows ProgramData directory."},
	{block: blockAssessment, key: "enable_filescan_userprofiles", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan Windows User Profiles."},
	{block: blockAssessment, key: "enable_filescan_linux_path", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the Linux $PATH directories."},
	{block: blockAssessment, key: "enable_filescan_linux_home", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan Linux home directories."},
	{block: blockAssessment, key: "enable_filescan_macos_path", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan the macOS $PATH directories."},
	{block: blockAssessment, key: "enable_filescan_macos_users", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan macOS /Users directories."},
	{block: blockAssessment, key: "enable_filescan_macos_applications", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan macOS /Applications directory."},
	{block: blockAssessment, key: "enable_filescan_macos_library", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan macOS /Library directory."},

	// -- report ----------------------------------------------------------------
	{block: blockReport, key: "report_verbosity", kind: kindChoice, family: familyAny, choices: []string{"Verbose", "Quiet", "Normal"},
		description: "Specifies the level of verbosity in the scan report. 'Verbose' reports as much information as possible."},
	{block: blockReport, key: "report_superseded_patches", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to show missing patches that have been superseded by newer patches."},
	{block: blockReport, key: "silent_dependencies", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to hide results from plugins that ran only as dependencies of other plugins."},
	{block: blockReport, key: "reverse_lookup", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to designate hosts by their DNS name instead of IP address."},
	{block: blockReport, key: "log_live_hosts", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to display hosts that respond to ping in the report."},
	{block: blockReport, key: "display_unreachable_hosts", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to display unreachable hosts in the scan results."},
	{block: blockReport, key: "display_unicode_characters", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to display Unicode characters in plugin output (for example, usernames, application names, SSL certificates)."},
	{block: blockReport, key: "allow_post_scan_editing", kind: kindYesNo, family: familyAny,
		description: "Indicates whether users can delete findings from the results of scans that use this policy."},

	// -- advanced --------------------------------------------------------------
	{block: blockAdvanced, key: "vendor_unpatched", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to scan for unpatched vulnerabilities where no patches or mitigations are available (including \"Will Not Fix\" CVEs)."},
	{block: blockAdvanced, key: "ignore_repos", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to disable Red Hat repository correlation and strictly use package version checks only."},
	{block: blockAdvanced, key: "always_report_ssh_cmds", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to always report SSH commands in the scan results."},
	{block: blockAdvanced, key: "enable_plugin_debugging", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to enable plugin debugging."},
	{block: blockAdvanced, key: "debug_level", kind: kindChoice, family: familyAny, choices: []string{"1", "2", "3", "4"},
		description: "Specifies the debug log level. '1' is Basic Debugging, '2' is Advanced Debugging, '3' is Full Debugging, '4' is Unrestricted Debugging."},
	{block: blockAdvanced, key: "enable_plugin_list", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to enumerate launched plugins in the scan results."},
	{block: blockAdvanced, key: "audit_trail", kind: kindChoice, family: familyAny, choices: []string{"full", "partial", "none"},
		description: "Specifies the audit trail verbosity. 'full' captures all audit trail data, 'partial' captures only scan errors, 'none' disables audit trail."},
	{block: blockAdvanced, key: "advanced_mode", kind: kindChoice, family: familyAny, choices: []string{"Default", "Scan low bandwidth links", "Custom"},
		description: "Specifies the preconfigured advanced mode for the scan. When set to a value other than 'Custom', the system applies predefined performance and general settings."},
	{block: blockAdvanced, key: "safe_checks", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to enable safe checks, which disables plugins that may have adverse effects on the remote host."},
	{block: blockAdvanced, key: "stop_scan_on_disconnect", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to stop scanning hosts that become unresponsive during the scan (for example, due to power off, denial of service, or IDS blocking)."},
	{block: blockAdvanced, key: "slice_network_addresses", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to scan IP addresses in a random order rather than sequentially."},
	{block: blockAdvanced, key: "auto_accept_disclaimer", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to automatically accept detected SSH disclaimer prompts during credentialed scans."},
	{block: blockAdvanced, key: "host_tagging", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to create a unique identifier on hosts scanned using credentials."},
	{block: blockAdvanced, key: "scan.allow_multi_target", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to scan targets with multiple domain names in parallel."},
	{block: blockAdvanced, key: "reduce_connections_on_congestion", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to slow down the scan when network congestion is detected."},
	{block: blockAdvanced, key: "network_receive_timeout", kind: kindNumber, family: familyNetwork,
		description: "Specifies the network timeout in seconds for receiving data."},
	{block: blockAdvanced, key: "max_checks_per_host", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of simultaneous checks per host."},
	{block: blockAdvanced, key: "max_hosts_per_scan", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of simultaneous hosts per scan."},
	{block: blockAdvanced, key: "max_simult_tcp_sessions_per_host", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of concurrent TCP sessions per host."},
	{block: blockAdvanced, key: "max_simult_tcp_sessions_per_scan", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of concurrent TCP sessions per scan."},
	{block: blockAdvanced, key: "reuse_ssh_connections", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the scanner reuses SSH connections between plugins."},
	{block: blockAdvanced, key: "scan.host_timeout_minutes", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum scan time per host in minutes. Leave empty or omit for no timeout."},
	{block: blockAdvanced, key: "patch_audit_over_telnet", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to perform patch auditing over Telnet."},
	{block: blockAdvanced, key: "patch_audit_over_rsh", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to perform patch auditing over RSH."},
	{block: blockAdvanced, key: "patch_audit_over_rexec", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to perform patch auditing over rexec."},
	{block: blockAdvanced, key: "snmp_port", kind: kindNumber, family: familyNetwork, min: 1, max: 65535,
		description: "Specifies the SNMP port."},
	{block: blockAdvanced, key: "additional_snmp_port1", kind: kindNumber, family: familyNetwork, min: 1, max: 65535,
		description: "Specifies an additional SNMP port."},
	{block: blockAdvanced, key: "additional_snmp_port2", kind: kindNumber, family: familyNetwork, min: 1, max: 65535,
		description: "Specifies an additional SNMP port."},
	{block: blockAdvanced, key: "additional_snmp_port3", kind: kindNumber, family: familyNetwork, min: 1, max: 65535,
		description: "Specifies an additional SNMP port."},
	{block: blockAdvanced, key: "http_login_method", kind: kindText, family: familyNetwork,
		description: "Specifies the HTTP login method."},
	{block: blockAdvanced, key: "http_reauth_delay", kind: kindNumber, family: familyNetwork,
		description: "Specifies the HTTP re-authentication delay in seconds."},
	{block: blockAdvanced, key: "http_login_max_redir", kind: kindNumber, family: familyNetwork,
		description: "Specifies the maximum number of HTTP login redirects to follow."},
	{block: blockAdvanced, key: "http_login_invert_auth_regex", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to invert the HTTP authentication regex."},
	{block: blockAdvanced, key: "http_login_auth_regex_on_headers", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to apply the HTTP authentication regex on response headers."},
	{block: blockAdvanced, key: "http_login_auth_regex_nocase", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the HTTP authentication regex should be case-insensitive."},
	{block: blockAdvanced, key: "apm_force_updates", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to force plugin updates before the scan."},
	{block: blockAdvanced, key: "apm_update_timeout", kind: kindNumber, family: familyNetwork,
		description: "Specifies the plugin update timeout in minutes."},
	{block: blockAdvanced, key: "region_dfw_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the DFW (Dallas) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "region_ord_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the ORD (Chicago) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "region_iad_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the IAD (Virginia) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "region_lon_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the LON (London) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "region_syd_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the SYD (Sydney) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "region_hkg_pref_name", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether the HKG (Hong Kong) cloud scanner region is enabled."},
	{block: blockAdvanced, key: "aws_ui_region_type", kind: kindText, family: familyNetwork,
		description: "Specifies the AWS cloud scanner region type."},
	{block: blockAdvanced, key: "aws_use_https", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to use HTTPS for AWS API calls."},
	{block: blockAdvanced, key: "aws_verify_ssl", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to verify SSL certificates for AWS API calls."},
	{block: blockAdvanced, key: "never_send_win_creds_in_the_clear", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to prevent sending Windows credentials in cleartext."},
	{block: blockAdvanced, key: "dont_use_ntlmv1", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to disable NTLMv1 authentication."},
	{block: blockAdvanced, key: "start_remote_registry", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to start the remote registry service on Windows targets."},
	{block: blockAdvanced, key: "enable_admin_shares", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to enable administrative shares on Windows targets."},
	{block: blockAdvanced, key: "start_server_service", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to start the Server service on Windows targets."},
	{block: blockAdvanced, key: "ssh_port", kind: kindNumber, family: familyNetwork, min: 1, max: 65535,
		description: "Specifies the SSH port to use for connecting to targets."},
	{block: blockAdvanced, key: "ssh_client_banner", kind: kindText, family: familyNetwork,
		description: "Specifies the SSH client banner string."},
	{block: blockAdvanced, key: "attempt_least_privilege", kind: kindYesNo, family: familyNetwork,
		description: "Specifies whether to attempt least privilege mode for credential-based scanning."},
	{block: blockAdvanced, key: "trusted_cas", kind: kindText, family: familyNetwork,
		description: "Specifies trusted CA certificates (in PEM format) that the scan considers as trusted."},
	{block: blockAdvanced, key: "custom_find_timeout", kind: kindNumber, family: familyAny,
		description: "Specifies the timeout in seconds for the Unix find command. Defaults to 240 seconds."},
	{block: blockAdvanced, key: "max_compliance_check_timeout", kind: kindNumber, family: familyAny,
		description: "Specifies the maximum compliance check timeout in seconds. This timeout applies primarily to Windows and Unix command-based compliance check plugins and overrides all other timeout settings when available."},
	{block: blockAdvanced, key: "max_compliance_output_length_kb", kind: kindNumber, family: familyAny,
		description: "Specifies the maximum compliance output length in KB. Values exceeding this limit are truncated."},
	{block: blockAdvanced, key: "compliance_generate_gold_image", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to generate a gold image .audit file from the compliance scan results."},
	{block: blockAdvanced, key: "compliance_generate_xccdf_results", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to generate XCCDF result files from the compliance scan."},
	{block: blockAdvanced, key: "compliance_generate_json_results", kind: kindYesNo, family: familyAny,
		description: "Specifies whether to generate JSON result files from the compliance scan."},
	{block: blockAdvanced, key: "report_max_ports", kind: kindInteger, family: familyAgent,
		description: "Specifies the maximum number of ports to report in agent scan results."},
	{block: blockAdvanced, key: "use_tenable_utils", kind: kindYesNo, family: familyAgent,
		description: "Specifies whether the agent uses Tenable-supplied binaries for 'find' and 'unzip' instead of native OS commands."},
	{block: blockAdvanced, key: "compliance_wfc_performance", kind: kindText, family: familyAgent,
		description: "Specifies the performance mode for Windows File Contents compliance checks."},
	{block: blockAdvanced, key: "unixfileanalysis_disable_xdev", kind: kindYesNo, family: familyAgent,
		description: "Specifies whether to prevent the Unix file analysis from crossing filesystem boundaries (equivalent to the find command '-xdev' option)."},

	// -- agent -----------------------------------------------------------------
	{block: blockAgent, key: "refresh_reporting_type", kind: kindChoice, family: familyAgent, choices: []string{"scans", "days"},
		description: "Specifies how often the Tenable Agent reports unchanged info-level vulnerability findings."},
	{block: blockAgent, key: "refresh_reporting_frequency_scans", kind: kindInteger, family: familyAgent, min: 4, max: 20,
		description: "Specifies the number of scans after which the Tenable Agent reports all findings."},
	{block: blockAgent, key: "refresh_reporting_frequency_days", kind: kindInteger, family: familyAgent, min: 7, max: 90,
		description: "Specifies the number of days after which the Tenable Agent reports all findings."},
	{block: blockAgent, key: "disable_refresh_reporting", kind: kindYesNo, family: familyAgent,
		description: "Specifies whether to force a refresh of all info-level findings on the next scan."},
	{block: blockAgent, key: "staggered_start_mins", kind: kindInteger, family: familyAgent,
		description: "Specifies the maximum number of minutes over which to stagger agent scan starts."},
}
