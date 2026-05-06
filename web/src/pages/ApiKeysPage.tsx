import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, Alert, message } from "antd";
import { PlusOutlined, CopyOutlined, StopOutlined, CheckCircleOutlined } from "@ant-design/icons";

const { Title } = Typography;
const { Option } = Select;

interface ApiKey {
  id: number;
  name: string;
  key_prefix: string;
  allowed_route_ids: string;
  enabled: boolean;
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
  raw_key?: string;
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
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchKeys(); }, [fetchKeys]);

  const handleCreate = async () => {
    const values = await form.validateFields();
    const res = await fetch("/api/api-keys", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        name: values.name,
        description: values.description || "",
        allowed_route_ids: values.scope === "all"
          ? JSON.stringify({ type: "all" })
          : JSON.stringify({ type: "selected", route_ids: values.route_ids || [] }),
      }),
    });
    if (!res.ok) { message.error("Failed to create key"); return; }
    const key = await res.json();
    setNewKey(key);
    setCreateOpen(false);
    form.resetFields();
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

  const formatTime = (t: string | null) => t ? new Date(t).toLocaleString() : "-";

  const columns = [
    { title: "Name", dataIndex: "name", key: "name" },
    { title: "Prefix", dataIndex: "key_prefix", key: "key_prefix", render: (v: string) => <code>{v}...</code> },
    { title: "Scope", dataIndex: "allowed_route_ids", key: "scope", render: (v: string) => {
      try { const p = JSON.parse(v); return <Tag>{p.type}</Tag>; } catch { return <Tag>all</Tag>; }
    }},
    { title: "Created", dataIndex: "created_at", key: "created_at", render: formatTime },
    { title: "Expires", dataIndex: "expires_at", key: "expires_at", render: formatTime },
    { title: "Last Used", dataIndex: "last_used_at", key: "last_used_at", render: formatTime },
    { title: "Status", dataIndex: "enabled", key: "enabled", render: (v: boolean) => v ? <Tag color="green">Enabled</Tag> : <Tag color="red">Disabled</Tag> },
    {
      title: "Actions", key: "actions",
      render: (_: unknown, record: ApiKey) => (
        <Space>
          {record.enabled ? (
            <Popconfirm title="Disable this key?" onConfirm={() => handleDisable(record.id)}>
              <Button size="small" icon={<StopOutlined />}>Disable</Button>
            </Popconfirm>
          ) : (
            <Popconfirm title="Enable this key?" onConfirm={() => handleEnable(record.id)}>
              <Button size="small" icon={<CheckCircleOutlined />}>Enable</Button>
            </Popconfirm>
          )}
          <Popconfirm title="Delete this key permanently?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>Delete</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: "space-between", width: "100%" }}>
        <Title level={3} style={{ margin: 0 }}>API Keys</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          Create API Key
        </Button>
      </Space>

      <Table columns={columns} dataSource={keys} rowKey="id" loading={loading} />

      <Modal title="Create API Key" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={handleCreate} okText="Create">
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input /></Form.Item>
          <Form.Item name="scope" label="Access" initialValue="all">
            <Select>
              <Option value="all">All models</Option>
              <Option value="selected">Selected routes</Option>
            </Select>
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
        <Input.TextArea
          value={newKey?.raw_key || ""}
          readOnly
          rows={2}
          style={{ fontFamily: "monospace", marginBottom: 8 }}
        />
        <Button icon={<CopyOutlined />} size="small" onClick={() => { navigator.clipboard.writeText(newKey?.raw_key || ""); message.success("Copied"); }}>
          Copy
        </Button>
      </Modal>
    </div>
  );
}

export default ApiKeysPage;
