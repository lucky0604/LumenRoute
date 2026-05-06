import { useState, useEffect, useCallback } from "react";
import { Typography, Table, Button, Modal, Form, Input, Select, Space, Tag, Popconfirm } from "antd";
import { PlusOutlined, CheckCircleOutlined } from "@ant-design/icons";

const { Title } = Typography;

interface Provider {
  id: number;
  name: string;
  description: string;
  provider_type: string;
  engine: string;
  base_url: string;
  auth_mode: string;
  health_check_path: string;
  health_status: string;
  last_error: string;
  enabled: boolean;
  created_at: string;
}

function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchProviders = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/providers", { credentials: "include" });
      if (res.status === 401) { setProviders([]); return; }
      const data = await res.json();
      setProviders(Array.isArray(data) ? data : []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchProviders(); }, [fetchProviders]);

  const handleCreate = async () => {
    const values = await form.validateFields();
    const res = await fetch("/api/providers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ ...values, enabled: true }),
    });
    if (!res.ok) { return; }
    setCreateOpen(false);
    form.resetFields();
    fetchProviders();
  };

  const handleDelete = async (id: number) => {
    await fetch(`/api/providers/${id}`, { method: "DELETE", credentials: "include" });
    fetchProviders();
  };

  const handleCheck = async (id: number) => {
    await fetch(`/api/providers/${id}/check`, { method: "POST", credentials: "include" });
    fetchProviders();
  };

  const columns = [
    { title: "Name", dataIndex: "name", key: "name" },
    { title: "Engine", dataIndex: "engine", key: "engine" },
    { title: "Base URL", dataIndex: "base_url", key: "base_url", render: (v: string) => <code style={{ fontSize: 12 }}>{v}</code> },
    { title: "Health", dataIndex: "health_status", key: "health",
      render: (s: string) => s === "healthy"
        ? <Tag color="green">Healthy</Tag>
        : s === "unhealthy"
          ? <Tag color="red">Unhealthy</Tag>
          : <Tag>Unknown</Tag>,
    },
    { title: "Enabled", dataIndex: "enabled", key: "enabled",
      render: (v: boolean) => v ? <Tag color="green">On</Tag> : <Tag color="red">Off</Tag>,
    },
    { title: "Created", dataIndex: "created_at", key: "created_at",
      render: (t: string) => t ? new Date(t).toLocaleString() : "-",
    },
    {
      title: "Actions", key: "actions",
      render: (_: unknown, record: Provider) => (
        <Space>
          <Button size="small" icon={<CheckCircleOutlined />} onClick={() => handleCheck(record.id)}>Check</Button>
          <Popconfirm title="Delete this provider permanently?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger>Delete</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: "space-between", width: "100%" }}>
        <Title level={3} style={{ margin: 0 }}>Providers</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          Add Provider
        </Button>
      </Space>

      <Table columns={columns} dataSource={providers} rowKey="id" loading={loading} />

      <Modal title="Add Provider" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={handleCreate} okText="Create">
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: "Provider name is required" }]}>
            <Input placeholder="e.g. my-vllm-cluster" />
          </Form.Item>
          <Form.Item name="provider_type" label="Type" initialValue="openai">
            <Select>
              <Select.Option value="openai">OpenAI-compatible</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="engine" label="Engine" initialValue="vllm">
            <Select>
              <Select.Option value="vllm">vLLM</Select.Option>
              <Select.Option value="sglang">SGLang</Select.Option>
              <Select.Option value="ollama">Ollama</Select.Option>
              <Select.Option value="openai">OpenAI</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="base_url" label="Base URL" rules={[{ required: true, message: "Base URL is required" }]}>
            <Input placeholder="e.g. http://192.168.1.100:8000" />
          </Form.Item>
          <Form.Item name="auth_mode" label="Auth Mode" initialValue="none">
            <Select>
              <Select.Option value="none">None</Select.Option>
              <Select.Option value="api_key">API Key</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="health_check_path" label="Health Check Path">
            <Input placeholder="e.g. /health (default: /models)" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default ProvidersPage;
