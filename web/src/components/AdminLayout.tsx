import { useState } from "react";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { Layout, Menu, Button, theme } from "antd";
import {
  CloudServerOutlined,
  NodeIndexOutlined,
  KeyOutlined,
  FileSearchOutlined,
  HeartOutlined,
  LogoutOutlined,
  BarChartOutlined,
} from "@ant-design/icons";

const { Sider, Header, Content } = Layout;

const menuItems = [
  {
    key: "config",
    label: "Configuration",
    type: "group" as const,
    children: [
      { key: "/providers", icon: <CloudServerOutlined />, label: "Providers" },
      { key: "/routes", icon: <NodeIndexOutlined />, label: "Routes" },
      { key: "/api-keys", icon: <KeyOutlined />, label: "API Keys" },
    ],
  },
  {
    key: "ops",
    label: "Operations",
    type: "group" as const,
    children: [
      { key: "/model-performance", icon: <BarChartOutlined />, label: "Model Performance" },
      { key: "/request-logs", icon: <FileSearchOutlined />, label: "Request Logs" },
      { key: "/health", icon: <HeartOutlined />, label: "Health Summary" },
    ],
  },
];

function AdminLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = theme.useToken();

  const selectedKey = location.pathname;

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        theme="light"
        width={240}
        style={{ borderRight: `1px solid ${token.colorBorderSecondary}` }}
      >
        <div
          style={{
            height: 64,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            fontWeight: 600,
            fontSize: collapsed ? 18 : 18,
            color: token.colorPrimary,
          }}
        >
          {collapsed ? "LR" : "LumenRoute"}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            padding: "0 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            height: 64,
          }}
        >
          <Button icon={<LogoutOutlined />} type="text" onClick={() => navigate("/login")}>
            Sign out
          </Button>
        </Header>
        <Content
          style={{
            background: token.colorBgLayout,
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}

export default AdminLayout;
