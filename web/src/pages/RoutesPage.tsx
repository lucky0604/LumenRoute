import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Typography, Table, Button, Modal, Form, Input, InputNumber, Space, Popconfirm, Drawer, Select, Switch, Tag, Empty, Alert } from "antd";
import { PlusOutlined, SettingOutlined, SearchOutlined } from "@ant-design/icons";
import StatusChip from "../components/StatusChip";

const { Title, Text } = Typography;

interface Provider {
  id: number; name: string; base_url: string; enabled: boolean; health_status: string;
}
interface SimpleProject {
  id: number; name: string;
}
interface Route {
  id: number; name: string; public_model_name: string; description: string;
  enabled: boolean; require_auth: boolean; project_id: number | null; project_name: string; created_at: string;
}
interface Target {
  id: number; route_id: number; provider_id: number; provider_name: string;
  upstream_model_name: string; weight: number; timeout_seconds: number;
  enabled: boolean; provider_healthy: boolean; provider_health_status: string;
}

function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [projects, setProjects] = useState<SimpleProject[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [routeOpen, setRouteOpen] = useState(false);
  const [targetOpen, setTargetOpen] = useState(false);
  const [selectedRoute, setSelectedRoute] = useState<Route | null>(null);
  const [targets, setTargets] = useState<Target[]>([]);
  const [routeForm] = Form.useForm();
  const [targetForm] = Form.useForm();
  const navigate = useNavigate();

  const fetchRoutes = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/routes", { credentials: "include" });
      if (res.ok) setRoutes(await res.json());
      else setError(`Failed to load routes (HTTP ${res.status})`);
    } catch { setError("Network error loading routes"); }
    finally { setLoading(false); }
  }, []);

  const fetchProjects = useCallback(async () => {
    const res = await fetch("/api/projects", { credentials: "include" });
    if (res.ok) {
      const data = await res.json();
      setProjects(data.map((p: SimpleProject) => ({ id: p.id, name: p.name })));
    }
  }, []);

  useEffect(() => { fetchRoutes(); fetchProjects(); }, [fetchRoutes, fetchProjects]);

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
    { title: "Public Model", dataIndex: "public_model_name", key: "model", render: (v: string) => <span className="mono">{v}</span> },
    { title: "Project", dataIndex: "project_name", key: "project", render: (v: string) => v ? <Tag>{v}</Tag> : <Text type="secondary">None</Text> },
    { title: "Require Auth", dataIndex: "require_auth", key: "auth", render: (v: boolean) => v ? <StatusChip label="Auth required" /> : null },
    { title: "Status", dataIndex: "enabled", key: "enabled", render: (v: boolean) => <StatusChip label={v ? "Enabled" : "Disabled"} /> },
    { title: "Created", dataIndex: "created_at", key: "created_at", render: (t: string) => new Date(t).toLocaleString() },
    { title: "Actions", key: "actions", render: (_: unknown, r: Route) => (
      <Space>
        <Button size="small" icon={<SettingOutlined />} onClick={() => openDrawer(r)}>Targets</Button>
        <Popconfirm title="Delete this route permanently?" onConfirm={async () => { await fetch(`/api/routes/${r.id}`, { method: "DELETE", credentials: "include" }); fetchRoutes(); }}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      </Space>
    )},
  ];

  const tcols = [
    { title: "Provider", dataIndex: "provider_name", key: "provider" },
    { title: "Upstream Model", dataIndex: "upstream_model_name", key: "model", render: (v: string) => <span className="mono">{v}</span> },
    { title: "Weight", dataIndex: "weight", key: "weight" },
    { title: "Timeout (s)", dataIndex: "timeout_seconds", key: "timeout" },
    { title: "Status", key: "status", render: (_: unknown, t: Target) => {
      if (!t.enabled) return <StatusChip label="Disabled" />;
      if (!t.provider_name) return <StatusChip label="Unhealthy" />;
      if (t.provider_health_status === "unknown") return <StatusChip label="Unverified" />;
      if (t.provider_health_status === "unhealthy") return <StatusChip label="Unhealthy" />;
      return <StatusChip label="Healthy" />;
    }},
    { title: "", key: "diagnose", width: 90, render: (_: unknown, t: Target) => (
      <Button size="small" type="link" icon={<SearchOutlined />} onClick={(e) => { e.stopPropagation(); navigate(`/diagnostics/targets/${t.id}`); }}>Diagnose</Button>
    )},
    { title: "", key: "act", render: (_: unknown, t: Target) => (
      <Popconfirm title="Delete this target permanently?" onConfirm={async () => { await fetch(`/api/route-targets/${t.id}`, { method: "DELETE", credentials: "include" }); if (selectedRoute) fetchTargets(selectedRoute.id); }}>
        <Button size="small" danger>Delete</Button>
      </Popconfirm>
    )},
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Routes</Title>
          <Text type="secondary">Define public model names and map them to upstream targets.</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setRouteOpen(true)}>Add Route</Button>
      </div>

      {error && <Alert title={error} type="error" showIcon closable style={{ marginBottom: 16 }} onClose={() => setError(null)} />}
      <div className="table-wrapper">
        <Table columns={columns} dataSource={routes} rowKey="id" loading={loading}
          locale={{ emptyText: <Empty description="No routes configured. Add a route to expose a public model name." /> }} />
      </div>

      <Modal title="Add Route" open={routeOpen} onCancel={() => { setRouteOpen(false); routeForm.resetFields(); }} onOk={async () => {
        const v = await routeForm.validateFields();
        await fetch("/api/routes", { method: "POST", headers: { "Content-Type": "application/json" }, credentials: "include", body: JSON.stringify({ ...v, enabled: true, require_auth: v.require_auth ?? true }) });
        setRouteOpen(false); routeForm.resetFields(); fetchRoutes();
      }}>
        <Form form={routeForm} layout="vertical">
          <Form.Item name="name" label="Route Name" rules={[{ required: true }]} extra="Display name for admin identification.">
            <Input placeholder="e.g. Qwen3.5-27B" />
          </Form.Item>
          <Form.Item name="public_model_name" label="Public Model Name" rules={[{ required: true }]} extra="Model name clients send to LumenRoute.">
            <Input placeholder="e.g. qwen-coder-fast" />
          </Form.Item>
          <Form.Item name="description" label="Description"><Input /></Form.Item>
          <Form.Item name="project_id" label="Project" extra="Optionally associate this route with a project for capture.">
            <Select placeholder="None" allowClear options={projects.map(p => ({ label: p.name, value: p.id }))} />
          </Form.Item>
          <Form.Item name="require_auth" label="Require API Key Auth" valuePropName="checked" initialValue={true} extra="When enabled, clients must provide a valid API key.">
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
          <Form.Item name="provider_id" label="Provider" rules={[{ required: true }]}>
            <Select placeholder="Select a provider" options={providers.map(p => ({ label: `${p.name} (${p.base_url})`, value: p.id }))} />
          </Form.Item>
          <Form.Item name="upstream_model_name" label="Upstream Model Name" rules={[{ required: true }]} extra="Actual model name sent to the selected provider.">
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
