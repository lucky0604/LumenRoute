import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Button, Modal, Form, Input, InputNumber, Space, Tag, Popconfirm, Drawer, Select, Switch } from "antd";
import { PlusOutlined, SettingOutlined } from "@ant-design/icons";

const { Title } = Typography;

interface Provider {
  id: number; name: string; base_url: string; enabled: boolean; health_status: string;
}
interface Route {
  id: number; name: string; public_model_name: string; description: string;
  enabled: boolean; require_auth: boolean; created_at: string;
}
interface Target {
  id: number; route_id: number; provider_id: number; provider_name: string;
  upstream_model_name: string; weight: number; timeout_seconds: number;
  enabled: boolean; provider_healthy: boolean; provider_health_status: string;
}

function targetStatus(t: Target) {
  if (!t.enabled) return <Tag color="default">Disabled</Tag>;
  if (!t.provider_name) return <Tag color="red">Provider Deleted</Tag>;
  if (t.provider_health_status === "unknown") return <Tag color="blue">Pending Health Check</Tag>;
  if (t.provider_health_status === "unhealthy") return <Tag color="red">Provider Unhealthy</Tag>;
  return <Tag color="green">Ready</Tag>;
}

function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [routeOpen, setRouteOpen] = useState(false);
  const [targetOpen, setTargetOpen] = useState(false);
  const [selectedRoute, setSelectedRoute] = useState<Route | null>(null);
  const [targets, setTargets] = useState<Target[]>([]);
  const [routeForm] = Form.useForm();
  const [targetForm] = Form.useForm();

  const fetchRoutes = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/routes", { credentials: "include" }); if (res.ok) setRoutes(await res.json()); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchRoutes(); }, [fetchRoutes]);

  const fetchTargets = async (routeId: number) => {
    const res = await fetch(`/api/routes/${routeId}/targets`, { credentials: "include" });
    if (res.ok) setTargets(await res.json());
  };

  const fetchProviders = async () => {
    const res = await fetch("/api/providers", { credentials: "include" });
    if (res.ok) setProviders(await res.json());
  };

  const openDrawer = async (r: Route) => {
    setSelectedRoute(r);
    await Promise.all([fetchTargets(r.id), fetchProviders()]);
  };

  const columns = [
    { title: "Name", dataIndex: "name", key: "name" },
    { title: "Public Model Name", dataIndex: "public_model_name", key: "model" },
    { title: "Require Auth", dataIndex: "require_auth", key: "auth", render: (v: boolean) => v ? <Tag color="blue">Yes</Tag> : <Tag>No</Tag> },
    { title: "Status", dataIndex: "enabled", key: "enabled", render: (v: boolean) => v ? <Tag color="green">Enabled</Tag> : <Tag color="red">Disabled</Tag> },
    { title: "Created", dataIndex: "created_at", key: "created_at", render: (t: string) => new Date(t).toLocaleString() },
    { title: "Actions", key: "actions", render: (_: unknown, r: Route) => (
      <Space>
        <Button size="small" icon={<SettingOutlined />} onClick={() => openDrawer(r)}>Targets</Button>
        <Popconfirm title="Delete this route?" onConfirm={async () => { await fetch(`/api/routes/${r.id}`, { method: "DELETE", credentials: "include" }); fetchRoutes(); }}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      </Space>
    )},
  ];

  const tcols = [
    { title: "Provider", dataIndex: "provider_name", key: "provider" },
    { title: "Upstream Model", dataIndex: "upstream_model_name", key: "model" },
    { title: "Weight", dataIndex: "weight", key: "weight" },
    { title: "Timeout (s)", dataIndex: "timeout_seconds", key: "timeout" },
    { title: "Status", key: "status", render: (_: unknown, t: Target) => targetStatus(t) },
    { title: "", key: "act", render: (_: unknown, t: Target) => (
      <Popconfirm title="Delete this target?" onConfirm={async () => { await fetch(`/api/route-targets/${t.id}`, { method: "DELETE", credentials: "include" }); if (selectedRoute) fetchTargets(selectedRoute.id); }}>
        <Button size="small" danger>Delete</Button>
      </Popconfirm>
    )},
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: "space-between", width: "100%" }}>
        <Title level={3} style={{ margin: 0 }}>Routes</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setRouteOpen(true)}>Add Route</Button>
      </Space>
      <Table columns={columns} dataSource={routes} rowKey="id" loading={loading} />

      <Modal title="Add Route" open={routeOpen} onCancel={() => { setRouteOpen(false); routeForm.resetFields(); }} onOk={async () => {
        const v = await routeForm.validateFields();
        await fetch("/api/routes", { method: "POST", headers: { "Content-Type": "application/json" }, credentials: "include", body: JSON.stringify({ ...v, enabled: true, require_auth: v.require_auth ?? true }) });
        setRouteOpen(false); routeForm.resetFields(); fetchRoutes();
      }}>
        <Form form={routeForm} layout="vertical">
          <Form.Item name="name" label="Route Name" rules={[{ required: true, message: "Required" }]}
            extra="Display name for admin identification, e.g. Qwen3.5-27B">
            <Input placeholder="e.g. Qwen3.5-27B" />
          </Form.Item>
          <Form.Item name="public_model_name" label="Public Model Name" rules={[{ required: true, message: "Required" }]}
            extra="External model name clients send to LumenRoute. This can be different from the upstream model name, e.g. qwen-coder-fast">
            <Input placeholder="e.g. qwen-coder-fast" />
          </Form.Item>
          <Form.Item name="description" label="Description"><Input /></Form.Item>
          <Form.Item name="require_auth" label="Require API Key Auth" valuePropName="checked" initialValue={true}
            extra="When enabled, API callers must provide a valid API key to access this route">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title={`Targets: ${selectedRoute?.name || ""}`} open={!!selectedRoute} onClose={() => { setSelectedRoute(null); setTargets([]); }} size={640}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setTargetOpen(true)} style={{ marginBottom: 16 }}>Add Target</Button>
        <Table columns={tcols} dataSource={targets} rowKey="id" size="small" />
      </Drawer>

      <Modal title="Add Target" open={targetOpen} onCancel={() => { setTargetOpen(false); targetForm.resetFields(); }} onOk={async () => {
        const v = await targetForm.validateFields();
        await fetch(`/api/routes/${selectedRoute?.id}/targets`, { method: "POST", headers: { "Content-Type": "application/json" }, credentials: "include", body: JSON.stringify({ ...v, enabled: true }) });
        setTargetOpen(false); targetForm.resetFields(); if (selectedRoute) fetchTargets(selectedRoute.id);
      }}>
        <Form form={targetForm} layout="vertical">
          <Form.Item name="provider_id" label="Provider" rules={[{ required: true, message: "Select a provider" }]}>
            <Select placeholder="Select a provider" options={providers.map(p => ({ label: `${p.name} (${p.base_url})`, value: p.id }))} />
          </Form.Item>
          <Form.Item name="upstream_model_name" label="Upstream Model Name" rules={[{ required: true, message: "Required" }]}
            extra="Actual model name sent to the selected provider. LumenRoute maps Public Model Name to this value, e.g. Qwen3.5-35B-A3B">
            <Input placeholder="e.g. Qwen3.5-35B-A3B" />
          </Form.Item>
          <Form.Item name="weight" label="Weight" initialValue={100}><InputNumber min={1} style={{ width: "100%" }} /></Form.Item>
          <Form.Item name="timeout_seconds" label="Timeout (s)" initialValue={120}><InputNumber min={1} style={{ width: "100%" }} /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default RoutesPage;
