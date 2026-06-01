import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Typography, Table, Button, Modal, Form, Input, InputNumber, Select, Space, Popconfirm, Switch, Tag } from "antd";
import { PlusOutlined, ExperimentOutlined } from "@ant-design/icons";
import StatusChip from "../components/StatusChip";

const { Title, Text } = Typography;

interface Project {
  id: number;
  name: string;
  description: string;
  data_category: string;
  capture_enabled: boolean;
  sample_rate: number;
  retention_days: number;
  has_export_token: boolean;
  routes_count: number;
  created_at: string;
  updated_at: string;
}

const categoryColors: Record<string, string> = {
  chat: "blue", code: "purple", translation: "cyan",
  summarization: "green", embedding: "orange", mixed: "default",
};

function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  const fetchProjects = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/projects", { credentials: "include" });
      if (res.ok) setProjects(await res.json());
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchProjects(); }, [fetchProjects]);

  const handleSave = async () => {
    const values = await form.validateFields();
    const method = editingProject ? "PUT" : "POST";
    const url = editingProject ? `/api/projects/${editingProject.id}` : "/api/projects";
    await fetch(url, {
      method,
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(values),
    });
    setModalOpen(false);
    setEditingProject(null);
    form.resetFields();
    fetchProjects();
  };

  const openEdit = (p: Project) => {
    setEditingProject(p);
    form.setFieldsValue(p);
    setModalOpen(true);
  };

  const openCreate = () => {
    setEditingProject(null);
    form.resetFields();
    form.setFieldsValue({ data_category: "mixed", sample_rate: 1.0, retention_days: 30, capture_enabled: false });
    setModalOpen(true);
  };

  const toggleCapture = async (p: Project, enabled: boolean) => {
    await fetch(`/api/projects/${p.id}`, {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...p, capture_enabled: enabled }),
    });
    fetchProjects();
  };

  const columns = [
    { title: "Name", dataIndex: "name", key: "name",
      render: (v: string, r: Project) => <a onClick={() => navigate(`/projects/${r.id}`)}>{v}</a> },
    { title: "Category", dataIndex: "data_category", key: "cat",
      render: (v: string) => <Tag color={categoryColors[v] || "default"}>{v}</Tag> },
    { title: "Capture", key: "capture", render: (_: unknown, r: Project) => (
      <Popconfirm title={`${r.capture_enabled ? "Disable" : "Enable"} capture?`} onConfirm={() => toggleCapture(r, !r.capture_enabled)}>
        <Switch checked={r.capture_enabled} size="small" />
      </Popconfirm>
    )},
    { title: "Sample Rate", dataIndex: "sample_rate", key: "rate",
      render: (v: number) => `${(v * 100).toFixed(0)}%` },
    { title: "Routes", dataIndex: "routes_count", key: "routes" },
    { title: "Export Token", key: "token",
      render: (_: unknown, r: Project) => r.has_export_token ? <StatusChip label="Configured" /> : <Text type="secondary">None</Text> },
    { title: "Actions", key: "actions", render: (_: unknown, r: Project) => (
      <Space>
        <Button size="small" onClick={() => openEdit(r)}>Edit</Button>
        <Popconfirm title="Delete this project? Associated routes will be disassociated." onConfirm={async () => {
          await fetch(`/api/projects/${r.id}`, { method: "DELETE", credentials: "include" });
          fetchProjects();
        }}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      </Space>
    )},
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Projects</Title>
          <Text type="secondary">Organize routes and manage data capture for training exports</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>New Project</Button>
      </div>
      <Table columns={columns} dataSource={projects} rowKey="id" loading={loading} pagination={{ pageSize: 20 }} />
      <Modal
        title={editingProject ? "Edit Project" : "New Project"}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => { setModalOpen(false); setEditingProject(null); }}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: "Name is required" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="data_category" label="Data Category">
            <Select options={[
              { value: "chat", label: "Chat" },
              { value: "code", label: "Code" },
              { value: "translation", label: "Translation" },
              { value: "summarization", label: "Summarization" },
              { value: "embedding", label: "Embedding" },
              { value: "mixed", label: "Mixed" },
            ]} />
          </Form.Item>
          <Form.Item name="capture_enabled" label="Capture Enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="sample_rate" label="Sample Rate" rules={[{ type: "number", min: 0, max: 1 }]}>
            <InputNumber step={0.1} min={0} max={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="retention_days" label="Retention (days)">
            <InputNumber min={1} max={365} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default ProjectsPage;
