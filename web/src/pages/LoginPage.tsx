import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, Form, Input, Button, Typography, Alert, Space } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

function LoginPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true);
    setError("");
    try {
      const formData = new URLSearchParams();
      formData.set("username", values.username);
      formData.set("password", values.password);
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: formData.toString(),
        credentials: "include",
      });
      if (!res.ok) {
        setError("Invalid credentials. Please check your username and password.");
        return;
      }
      navigate("/providers", { replace: true });
    } catch {
      setError("Unable to connect to server. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "100vh",
        background: "#F8FAFC",
      }}
    >
      <Card style={{ width: 400 }} styles={{ body: { padding: 32 } }}>
        <Space orientation="vertical" size="large" style={{ width: "100%" }}>
          <div style={{ textAlign: "center" }}>
            <Title level={3} style={{ marginBottom: 4 }}>LumenRoute</Title>
            <Text type="secondary">Internal model routing control plane</Text>
          </div>
          {error && <Alert type="error" title={error} showIcon />}
          <Form name="login" onFinish={onFinish} layout="vertical" size="large">
            <Form.Item name="username" rules={[{ required: true, message: "Username is required" }]}>
              <Input prefix={<UserOutlined />} placeholder="Username" autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" rules={[{ required: true, message: "Password is required" }]}>
              <Input.Password prefix={<LockOutlined />} placeholder="Password" autoComplete="current-password" />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={loading} block>
                Sign in
              </Button>
            </Form.Item>
          </Form>
        </Space>
      </Card>
    </div>
  );
}

export default LoginPage;
