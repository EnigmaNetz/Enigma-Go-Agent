Name:           enigma-agent
Version:        %{?_version:}%{_version}%{!?_version:0.1.0}
Release:        %{?_release:}%{_release}%{!?_release:1}
Summary:        Enigma Network Capture Agent
License:        Proprietary
URL:            https://getenigma.ai
Source0:        enigma-agent
Source1:        config.json

BuildArch:      %{?_arch:}%{_arch}%{!?_arch:x86_64}
Requires:       zeek, tcpdump

%description
A cross-platform network capture agent that collects, processes, and optionally uploads network traffic data in standardized Zeek-format logs.

%prep
# Nothing to prep

%build
# Nothing to build

%install
mkdir -p %{buildroot}/usr/local/bin
install -m 0755 %{SOURCE0} %{buildroot}/usr/local/bin/enigma-agent
mkdir -p %{buildroot}/etc/enigma-agent
install -m 0644 %{SOURCE1} %{buildroot}/etc/enigma-agent/config.json

# Systemd service
mkdir -p %{buildroot}/usr/lib/systemd/system
cat > %{buildroot}/usr/lib/systemd/system/enigma-agent.service <<EOF
[Unit]
Description=Enigma Network Capture Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/enigma-agent
Restart=on-failure
User=root

[Install]
WantedBy=multi-user.target
EOF

%files
/usr/local/bin/enigma-agent
/etc/enigma-agent/config.json
/usr/lib/systemd/system/enigma-agent.service

%post
if [ $1 -eq 1 ] ; then
    systemctl daemon-reload || true
    systemctl enable enigma-agent || true
    systemctl restart enigma-agent || true
fi

%preun
if [ $1 -eq 0 ] ; then
    systemctl stop enigma-agent || true
    systemctl disable enigma-agent || true
fi

%changelog
* $(date '+%a %b %d %Y') Enigma Team <support@getenigma.ai> - %{version}-%{release}
- Initial RPM release