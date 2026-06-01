import { Typography } from "antd";

const { Title, Text } = Typography;

function ControlCenterPage() {
  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>Control Center</Title>
      <Text type="secondary">Routing plane health overview.</Text>
    </div>
  );
}

export default ControlCenterPage;
