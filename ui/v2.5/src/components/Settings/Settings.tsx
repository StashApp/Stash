import React from "react";
import queryString from "query-string";
import { Card, Tab, Nav, Row, Col } from "react-bootstrap";
import { useHistory, useLocation } from "react-router-dom";
import { FormattedMessage } from "react-intl";
import { SettingsAboutPanel } from "./SettingsAboutPanel";
import { SettingsConfigurationPanel } from "./SettingsConfigurationPanel";
import { SettingsInterfacePanel } from "./SettingsInterfacePanel/SettingsInterfacePanel";
import { SettingsLogsPanel } from "./SettingsLogsPanel";
import { SettingsTasksPanel } from "./SettingsTasksPanel/SettingsTasksPanel";
import { SettingsPluginsPanel } from "./SettingsPluginsPanel";
import { SettingsScrapingPanel } from "./SettingsScrapingPanel";
import { SettingsToolsPanel } from "./SettingsToolsPanel";
import { SettingsDLNAPanel } from "./SettingsDLNAPanel";

export const Settings: React.FC = () => {
  const location = useLocation();
  const history = useHistory();
  const defaultTab = queryString.parse(location.search).tab ?? "tasks";

  const onSelect = (val: string) => history.push(`?tab=${val}`);

  return (
    <Card className="col col-lg-9 mx-auto">
      <Tab.Container
        activeKey={defaultTab}
        id="configuration-tabs"
        onSelect={(tab) => tab && onSelect(tab)}
      >
        <Row>
          <Col sm={3} md={2}>
            <Nav variant="pills" className="flex-column">
              <Nav.Item>
                <Nav.Link eventKey="configuration">
                  <FormattedMessage id="configuration" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="interface">
                  <FormattedMessage id="config.categories.interface" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="tasks">
                  <FormattedMessage id="config.categories.tasks" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="dlna">DLNA</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="tools">
                  <FormattedMessage id="config.categories.tools" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="scraping">
                  <FormattedMessage id="config.categories.scraping" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="plugins">
                  <FormattedMessage id="config.categories.plugins" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="logs">
                  <FormattedMessage id="config.categories.logs" />
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="about">
                  <FormattedMessage id="config.categories.about" />
                </Nav.Link>
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
              <Tab.Pane eventKey="dlna" unmountOnExit>
                <SettingsDLNAPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="tools" unmountOnExit>
                <SettingsToolsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="scraping" unmountOnExit>
                <SettingsScrapingPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="plugins" unmountOnExit>
                <SettingsPluginsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="logs" unmountOnExit>
                <SettingsLogsPanel />
              </Tab.Pane>
              <Tab.Pane eventKey="about" unmountOnExit>
                <SettingsAboutPanel />
              </Tab.Pane>
            </Tab.Content>
          </Col>
        </Row>
      </Tab.Container>
    </Card>
  );
};
