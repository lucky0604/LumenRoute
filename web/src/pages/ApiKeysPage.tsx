import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Button, Modal, Form, Input, Select, Space, Popconfirm, Alert, message } from "antd";
import { PlusOutlined, CopyOutlined, StopOutlined, CheckCircleOutlined } from "@ant-design/icons";
import StatusChip from "../components/StatusChip";
import EmptyState from "../components/EmptyState";

const { Title, Text } = Typography;

interface ApiKey {
  id: number; name: string; key_prefix: string; allowed_route_ids: string;
  enabled: boolean; expires_at: string | null; last_used_at: string | null;
  created_at: string; raw_key?: string;
}

function ApiKeysPage() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [newKey, setNewKey] = useState<ApiKey | null>(null);
  const [form] = Form.useForm();

  const fetchKeys = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/api-keys", { credentials: "include" });
      if (res.status === 401) { setKeys([]); return; }
      const data = await res.json();
      setKeys(Array.isArray(data) ? data : []);
    } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchKeys(); }, [fetchKeys]);

  const handleCreate = async () => {
    const values = await form.validateFields();
    const res = await fetch("/api/api-keys", {
      method: "POST", headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        name: values.name, description: values.description || "",
        allowed_route_ids: values.scope === "all"
          ? JSON.stringify({ type: "all" })
          : JSON.stringify({ type: "selected", route_ids: values.route_ids || [] }),
      }),
    });
    if (!res.ok) { message.error("Failed to create key"); return; }
    const key = await res.json();
    setNewKey(key); setCreateOpen(false); form.resetFields();
  };

  const handleDisable = async (id: number) => {
    await fetch(`/api/api-keys/${id}/disable`, { method: "POST", credentials: "include" });
    fetchKeys();
  };
  const handleEnable = async (id: number) => {
    await fetch(`/api/api-keys/${id}/enable`, { method: "POST", credentials: "include" });
    fetchKeys();
  };
  const handleDelete = async (id: number) => {
    await fetch(`/api/api-keys/${id}`, { method: "DELETE", credentials: "include" });
    fetchKeys();
  };

  const columns = [
    { title: "Name", dataIndex: "name", key: "name" },
    { title: "Prefix", dataIndex: "key_prefix", key: "key_prefix", render: (v: string) => <span className="mono">{v}...</span> },
    { title: "Scope", dataIndex: "allowed_route_ids", key: "scope", render: (v: string) => {
      try { const p = JSON.parse(v); return <StatusChip label={p.type === "all" ? "All access" : "Scoped"} />; } catch { return <StatusChip label="Scoped" />; }
    }},
    { title: "Created", dataIndex: "created_at", key: "created_at", render: (t: string) => new Date(t).toLocaleString() },
    { title: "Expires", dataIndex: "expires_at", key: "expires_at", render: (t: string | null) => t ? new Date(t).toLocaleString() : "-" },
    { title: "Last Used", dataIndex: "last_used_at", key: "last_used_at", render: (t: string | null) => t ? new Date(t).toLocaleString() : "-" },
    { title: "Status", dataIndex: "enabled", key: "enabled", render: (v: boolean) => <StatusChip label={v ? "Enabled" : "Disabled"} /> },
    { title: "Actions", key: "actions", render: (_: unknown, r: ApiKey) => (
      <Space>
        {r.enabled ? (
          <Popconfirm title="Disable this key?" onConfirm={() => handleDisable(r.id)}>
            <Button size="small" icon={<StopOutlined />}>Disable</Button>
          </Popconfirm>
        ) : (
          <Popconfirm title="Enable this key?" onConfirm={() => handleEnable(r.id)}>
            <Button size="small" icon={<CheckCircleOutlined />}>Enable</Button>
          </Popconfirm>
        )}
        <Popconfirm title="Delete this key permanently?" onConfirm={() => handleDelete(r.id)}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      </Space>
    )},
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>API Keys</Title>
          <Text type="secondary">Issue and manage proxy access credentials.</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>Create API Key</Button>
      </div>

      <div className="table-wrapper">
        <Table
          columns={columns}
          dataSource={keys}
          rowKey="id"
          loading={loading}
          locale={{ emptyText: <EmptyState reason="No API keys yet. Create a key to allow clients to call LumenRoute routes." action="Create API Key" onAction={() => setCreateOpen(true)} /> }}
        />
      </div>

      <Modal title="Create API Key" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={handleCreate} okText="Create">
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input /></Form.Item>
          <Form.Item name="scope" label="Access" initialValue="all">
            <Select options={[{ label: "All models", value: "all" }, { label: "Selected routes", value: "selected" }]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="API Key Created"
        open={!!newKey}
        onCancel={() => setNewKey(null)}
        footer={[<Button key="done" type="primary" onClick={() => setNewKey(null)}>I have saved this key</Button>]}
        closable={false}
        maskClosable={false}
      >
        <Alert type="warning" title="Store this key now. It will not be shown again." style={{ marginBottom: 16 }} showIcon />
        <Input.TextArea value={newKey?.raw_key || ""} readOnly rows={2} className="mono" style={{ marginBottom: 8 }} />
        <Button
          icon={<CopyOutlined />}
          size="small"
          onClick={() => { navigator.clipboard.writeText(newKey?.raw_key || ""); message.success("Key copied"); }}
          aria-label="Copy API key secret"
        >
          Copy
        </Button>
      </Modal>
    </div>
  );
}

export default ApiKeysPage;
