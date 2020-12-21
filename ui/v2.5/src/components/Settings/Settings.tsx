import React from "react";
import queryString from "query-string";
import { Card, Tab, Nav, Row, Col } from "react-bootstrap";
import { useHistory, useLocation } from "react-router-dom";
import { SettingsAboutPanel } from "./SettingsAboutPanel";
import { SettingsConfigurationPanel } from "./SettingsConfigurationPanel";
import { SettingsInterfacePanel } from "./SettingsInterfacePanel/SettingsInterfacePanel";
import { SettingsLogsPanel } from "./SettingsLogsPanel";
import { SettingsTasksPanel } from "./SettingsTasksPanel/SettingsTasksPanel";
import { SettingsPluginsPanel } from "./SettingsPluginsPanel";
import { SettingsScrapersPanel } from "./SettingsScrapersPanel";
import { SettingsWarningsPanel } from "./SettingsWarningsPanel";

export const Settings: React.FC = () => {
  const location = useLocation();
  const history = useHistory();
  const defaultTab = queryString.parse(location.search).tab ?? "tasks";

  const onSelect = (val: string) => history.push(`?tab=${val}`);

  return (
    <Card className="col col-lg-9 mx-auto">
      <Tab.Container
        defaultActiveKey={defaultTab}
        id="configuration-tabs"
        onSelect={(tab) => tab && onSelect(tab)}
      >
        <Row>
          <Col sm={3} md={2}>
            <Nav variant="pills" className="flex-column">
              <Nav.Item>
                <Nav.Link eventKey="configuration">Configuration</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="interface">Interface</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="tasks">Tasks</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="scrapers">Scrapers</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="plugins">Plugins</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="logs">Logs</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="warnings">Warnings</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="about">About</Nav.Link>
              </Nav.Item>
              <hr className="d-sm-none" />
            </Nav>
          </Col>
          <Col sm={9} md={10}>
            <Tab.Content>
              <Tab.Pane eventKey="configuration">
                <SettingsConfigurationPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="interface">
                <SettingsInterfacePanel />
              </Tab.Pane>
              <Tab.Pane eventKey="tasks">
                <SettingsTasksPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="scrapers">
                <SettingsScrapersPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="plugins">
                <SettingsPluginsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="logs">
                <SettingsLogsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="warnings">
                <SettingsWarningsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="about">
                <SettingsAboutPanel />
              </Tab.Pane>
            </Tab.Content>
          </Col>
        </Row>
      </Tab.Container>
    </Card>
  );
};
