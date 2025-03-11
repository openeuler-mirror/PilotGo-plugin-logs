%define         debug_package %{nil}

Name:           PilotGo-plugin-logs
Version:        1.0.0
Release:        1
Summary:        logs plugin for PilotGo
License:        MulanPSL-2.0
URL:            https://gitee.com/openeuler/PilotGo-plugin-logs
Source0:        https://gitee.com/src-openeuler/PilotGo-plugin-logs/%{name}-%{version}.tar.gz

ExclusiveArch: x86_64 aarch64

BuildRequires:  systemd
BuildRequires:  golang
BuildRequires:  nodejs
BuildRequires:  npm

%description
logs plugin for PilotGo

%package        server
Summary:        PilotGo-plugin-logs server
Provides:       pilotgo-plugin-logs-server = %{version}-%{release}

%description    server
PilotGo-plugin-logs server.

%package        agent
Summary:        PilotGo-plugin-logs agent
Provides:       pilotgo-plugin-logs-agent = %{version}-%{release}

%description    agent
PilotGo-plugin-logs agent.

%prep
%autosetup -p1 -n %{name}-%{version}

%build
# web
pushd web
npm run install
npm run build
popd
cp -rf web/dist/* cmd/server/webserver/frontendResource/
# server
pushd cmd/server
GOWORK=off GO111MODULE=on go build -tags=production -o PilotGo-plugin-logs-server main.go
popd
# agent
pushd cmd/agent
GOWORK=off GO111MODULE=on go build -o PilotGo-plugin-logs-agent main.go
popd

%install
mkdir -p %{buildroot}/opt/PilotGo/plugin/logs/server/log
mkdir -p %{buildroot}/opt/PilotGo/plugin/logs/agent/log
# server
install -D -m 0755 %{_builddir}/PilotGo-plugin-logs/cmd/server/PilotGo-plugin-logs-server %{buildroot}/opt/PilotGo/plugin/logs/server
install -D -m 0644 %{_builddir}/PilotGo-plugin-logs/cmd/server/logs_server.yaml.template %{buildroot}/opt/PilotGo/plugin/logs/server/logs_server.yaml
install -D -m 0644 %{_builddir}/PilotGo-plugin-logs/scripts/PilotGo-plugin-logs-server.service %{buildroot}%{_unitdir}/PilotGo-plugin-logs-server.service
# agent
install -D -m 0755 %{_builddir}/PilotGo-plugin-logs/cmd/agent/PilotGo-plugin-logs-agent %{buildroot}/opt/PilotGo/plugin/logs/agent
install -D -m 0644 %{_builddir}/PilotGo-plugin-logs/cmd/agent/logs_agent.yaml.template %{buildroot}/opt/PilotGo/plugin/logs/agent/logs_agent.yaml
install -D -m 0644 %{_builddir}/PilotGo-plugin-logs/scripts/PilotGo-plugin-logs-agent.service %{buildroot}%{_unitdir}/PilotGo-plugin-logs-agent.service

%files          server
%dir /opt/PilotGo
%dir /opt/PilotGo/plugin
%dir /opt/PilotGo/plugin/logs
%dir /opt/PilotGo/plugin/logs/server
%dir /opt/PilotGo/plugin/logs/server/log
/opt/PilotGo/plugin/logs/server/PilotGo-plugin-logs-server
/opt/PilotGo/plugin/logs/server/logs_server.yaml
%{_unitdir}/PilotGo-plugin-logs-server.service

%files          agent
%dir /opt/PilotGo
%dir /opt/PilotGo/plugin
%dir /opt/PilotGo/plugin/logs
%dir /opt/PilotGo/plugin/logs/agent
%dir /opt/PilotGo/plugin/logs/agent/log
/opt/PilotGo/plugin/logs/agent/PilotGo-plugin-logs-agent
/opt/PilotGo/plugin/logs/agent/logs_agent.yaml
%{_unitdir}/PilotGo-plugin-logs-agent.service

%changelog
* Fri Feb 28 2025 wangjunqi <wangjunqi@kylinos.cn> - 1.0.0-1
- initialize 
