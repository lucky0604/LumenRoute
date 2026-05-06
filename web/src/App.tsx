import { Routes, Route, Navigate } from "react-router-dom";
import { ConfigProvider, App as AntApp } from "antd";
import LoginPage from "./pages/LoginPage";
import ProvidersPage from "./pages/ProvidersPage";
import RoutesPage from "./pages/RoutesPage";
import ApiKeysPage from "./pages/ApiKeysPage";
import RequestLogsPage from "./pages/RequestLogsPage";
import HealthSummaryPage from "./pages/HealthSummaryPage";
import AdminLayout from "./components/AdminLayout";

const themeConfig = {
  token: {
    colorPrimary: "#2563EB",
    colorSuccess: "#10B981",
    colorWarning: "#F59E0B",
    colorError: "#EF4444",
    colorInfo: "#6366F1",
    colorBgLayout: "#F8FAFC",
    colorBgContainer: "#FFFFFF",
    colorBorder: "#E2E8F0",
    colorBorderSecondary: "#E2E8F0",
    colorText: "#0F172A",
    colorTextSecondary: "#64748B",
    borderRadius: 6,
    fontFamily: `Inter, "Fira Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,
  },
};

function App() {
  return (
    <ConfigProvider theme={themeConfig}>
      <AntApp>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<AdminLayout />}>
            <Route path="/providers" element={<ProvidersPage />} />
            <Route path="/routes" element={<RoutesPage />} />
            <Route path="/api-keys" element={<ApiKeysPage />} />
            <Route path="/request-logs" element={<RequestLogsPage />} />
            <Route path="/health" element={<HealthSummaryPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </AntApp>
    </ConfigProvider>
  );
}

export default App;
