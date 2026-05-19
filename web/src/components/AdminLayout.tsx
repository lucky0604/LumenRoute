import { useState } from "react";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { Layout, Menu, Button, theme, Grid } from "antd";
import {
  CloudServerOutlined,
  NodeIndexOutlined,
  KeyOutlined,
  FileSearchOutlined,
  HeartOutlined,
  LogoutOutlined,
  BarChartOutlined,
  SunOutlined,
  MoonOutlined,
  DashboardOutlined,
  BugOutlined,
  MenuOutlined,
} from "@ant-design/icons";
import type { ThemeMode } from "../theme/tokens";

const { Sider, Header, Content } = Layout;
const { useBreakpoint } = Grid;

const menuItems = [
  {
    key: "overview",
    label: "Overview",
    type: "group" as const,
    children: [
      { key: "/control-center", icon: <DashboardOutlined />, label: "Control Center" },
    ],
  },
  {
    key: "configure",
    label: "Configure",
    type: "group" as const,
    children: [
      { key: "/providers", icon: <CloudServerOutlined />, label: "Providers" },
      { key: "/routes", icon: <NodeIndexOutlined />, label: "Routes" },
      { key: "/api-keys", icon: <KeyOutlined />, label: "API Keys" },
    ],
  },
  {
    key: "observe",
    label: "Observe",
    type: "group" as const,
    children: [
      { key: "/request-logs", icon: <FileSearchOutlined />, label: "Request Logs" },
      { key: "/health", icon: <HeartOutlined />, label: "Health" },
      { key: "/model-performance", icon: <BarChartOutlined />, label: "Model Performance" },
    ],
  },
  {
    key: "diagnose",
    label: "Diagnose",
    type: "group" as const,
    children: [
      { key: "/model-performance", icon: <BugOutlined />, label: "Target Diagnosis" },
    ],
  },
];

interface AdminLayoutProps {
  themeMode: ThemeMode;
  onToggleTheme: () => void;
}

function AdminLayout({ themeMode, onToggleTheme }: AdminLayoutProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = theme.useToken();
  const screens = useBreakpoint();

  const isMobile = !screens.lg;
  const selectedKey = location.pathname;

  const onMenuClick = (key: string) => {
    navigate(key);
    if (isMobile) setMobileDrawerOpen(false);
  };

  const renderSider = () => (
    <Sider
      collapsible={!isMobile}
      collapsed={isMobile ? false : collapsed}
      onCollapse={isMobile ? undefined : setCollapsed}
      width={248}
      collapsedWidth={72}
      breakpoint="lg"
      trigger={null}
      style={{
        background: token.colorBgContainer,
        borderRight: `1px solid ${token.colorBorderSecondary}`,
      }}
    >
      <div
        style={{
          height: 64,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          fontWeight: 600,
          fontSize: collapsed && !isMobile ? 18 : 18,
          color: token.colorPrimary,
        }}
      >
        {collapsed && !isMobile ? "LR" : "LumenRoute"}
      </div>
      <Menu
        mode="inline"
        selectedKeys={[selectedKey]}
        items={menuItems}
        onClick={({ key }) => onMenuClick(key)}
        style={{ borderRight: 0, background: "transparent" }}
      />
    </Sider>
  );

  return (
    <Layout style={{ minHeight: "100vh" }}>
      {isMobile ? (
        mobileDrawerOpen && (
          <div
            style={{
              position: "fixed",
              inset: 0,
              zIndex: 1000,
              background: "rgba(0,0,0,0.45)",
            }}
            onClick={() => setMobileDrawerOpen(false)}
          >
            <div style={{ width: 248, height: "100%" }} onClick={(e) => e.stopPropagation()}>
              {renderSider()}
            </div>
          </div>
        )
      ) : (
        renderSider()
      )}
      <Layout>
        <Header
          style={{
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            padding: "0 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: isMobile ? "space-between" : "flex-end",
            height: 64,
            gap: 8,
          }}
        >
          {isMobile && (
            <Button
              icon={<MenuOutlined />}
              type="text"
              onClick={() => setMobileDrawerOpen(true)}
              aria-label="Toggle navigation"
            />
          )}
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <Button
              icon={themeMode === "dark" ? <SunOutlined /> : <MoonOutlined />}
              type="text"
              onClick={onToggleTheme}
              aria-label={`Switch to ${themeMode === "dark" ? "light" : "dark"} theme`}
            />
            <Button icon={<LogoutOutlined />} type="text" onClick={() => navigate("/login")}>
              Sign out
            </Button>
          </div>
        </Header>
        <Content
          id="main-content"
          style={{
            background: token.colorBgLayout,
            minHeight: 280,
            overflow: "auto",
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}

export default AdminLayout;
