import React, { Dispatch, useRef, useState } from "react";
import {
  Badge,
  Button,
  Card,
  Collapse,
  Form,
  InputGroup,
} from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Icon } from "src/components/Shared";
import { useConfiguration } from "src/core/StashService";
import { TextUtils } from "src/utils";
import FieldSelector from "../FieldSelector";

import { ITaggerConfig, ParseMode, SCENE_FIELDS } from "../constants";

interface IConfigProps {
  show: boolean;
  config: ITaggerConfig;
  setConfig: Dispatch<ITaggerConfig>;
}

const Config: React.FC<IConfigProps> = ({ show, config, setConfig }) => {
  const intl = useIntl();
  const stashConfig = useConfiguration();
  const blacklistRef = useRef<HTMLInputElement | null>(null);
  const [showExclusionModal, setShowExclusionModal] = useState(false);
  const excludedFields = config.excludedSceneFields ?? [];

  const handleInstanceSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedEndpoint = e.currentTarget.value;
    setConfig({
      ...config,
      selectedEndpoint,
    });
  };

  const removeBlacklist = (index: number) => {
    setConfig({
      ...config,
      blacklist: [
        ...config.blacklist.slice(0, index),
        ...config.blacklist.slice(index + 1),
      ],
    });
  };

  const handleBlacklistAddition = () => {
    if (!blacklistRef.current) return;

    const input = blacklistRef.current.value;
    if (input.length === 0) return;

    setConfig({
      ...config,
      blacklist: [...config.blacklist, input],
    });
    blacklistRef.current.value = "";
  };

  const stashBoxes = stashConfig.data?.configuration.general.stashBoxes ?? [];

  const handleFieldSelect = (fields: string[]) => {
    setConfig({ ...config, excludedSceneFields: fields });
    setShowExclusionModal(false);
  };

  return (
    <>
      <Collapse in={show}>
        <Card>
          <div className="row">
            <h4 className="col-12">
              <FormattedMessage id="configuration" />
            </h4>
            <hr className="w-100" />
            <Form className="col-md-6">
              <Form.Group controlId="excluded-scene-fields">
                <h6>Excluded fields:</h6>
                <span>
                  {excludedFields.length > 0
                    ? excludedFields.map((f) => (
                        <Badge variant="secondary" className="tag-item">
                          {TextUtils.capitalize(f)}
                        </Badge>
                      ))
                    : "No fields are excluded"}
                </span>
                <Form.Text>
                  These fields will not be changed when updating scenes.
                </Form.Text>
                <Button
                  onClick={() => setShowExclusionModal(true)}
                  className="mt-2"
                >
                  Edit Excluded Fields
                </Button>
              </Form.Group>

              <Form.Group controlId="tag-males" className="align-items-center">
                <Form.Check
                  label={
                    <FormattedMessage id="component_tagger.config.show_male_label" />
                  }
                  checked={config.showMales}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setConfig({ ...config, showMales: e.currentTarget.checked })
                  }
                />
                <Form.Text>
                  <FormattedMessage id="component_tagger.config.show_male_desc" />
                </Form.Text>
              </Form.Group>

              <Form.Group className="align-items-center">
                <div className="d-flex align-items-center">
                  <Form.Check
                    id="tag-mode"
                    label="Set tags"
                    className="mr-4"
                    checked={config.setTags}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setConfig({ ...config, setTags: e.currentTarget.checked })
                    }
                  />
                  <Form.Control
                    id="tag-operation"
                    className="col-md-2 col-3 input-control"
                    as="select"
                    value={config.tagOperation}
                    onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
                      setConfig({
                        ...config,
                        tagOperation: e.currentTarget.value,
                      })
                    }
                    disabled={!config.setTags}
                  >
                    <option value="merge">
                      {intl.formatMessage({ id: "actions.merge" })}
                    </option>
                    <option value="overwrite">
                      {intl.formatMessage({ id: "actions.overwrite" })}
                    </option>
                  </Form.Control>
                </div>
                <Form.Text>
                  <FormattedMessage id="component_tagger.config.set_tag_desc" />
                </Form.Text>
              </Form.Group>

              <Form.Group className="align-items-center">
                <div className="d-flex align-items-center">
                  <Form.Check
                    id="create-tags"
                    label="Create tags"
                    className="mr-4"
                    checked={config.createTags}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setConfig({
                        ...config,
                        createTags: e.currentTarget.checked,
                      })
                    }
                  />
                </div>
                <Form.Text>
                  Sets whether tags that don&apos;t exist in stash should be
                  created or ignored.
                </Form.Text>
              </Form.Group>

              <Form.Group className="align-items-center">
                <div className="d-flex align-items-center">
                  <Form.Check
                    id="set-organized"
                    label="Set organized flag on tagged scenes"
                    className="mr-4"
                    checked={config.setOrganized}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setConfig({
                        ...config,
                        setOrganized: e.currentTarget.checked,
                      })
                    }
                  />
                </div>
              </Form.Group>
            </Form>
            <div className="col-md-6">
              <h5>
                <FormattedMessage id="component_tagger.config.blacklist_label" />
              </h5>
              <InputGroup>
                <Form.Control className="text-input" ref={blacklistRef} />
                <InputGroup.Append>
                  <Button onClick={handleBlacklistAddition}>
                    <FormattedMessage id="actions.add" />
                  </Button>
                </InputGroup.Append>
              </InputGroup>
              <div>
                {intl.formatMessage(
                  { id: "component_tagger.config.blacklist_desc" },
                  { chars_require_escape: <code>[\^$.|?*+()</code> }
                )}
              </div>
              {config.blacklist.map((item, index) => (
                <Badge
                  className="tag-item d-inline-block"
                  variant="secondary"
                  key={item}
                >
                  {item.toString()}
                  <Button
                    className="minimal ml-2"
                    onClick={() => removeBlacklist(index)}
                  >
                    <Icon icon="times" />
                  </Button>
                </Badge>
              ))}

              <Form.Group controlId="mode-select">
                <div className="row no-gutters">
                  <Form.Label className="mr-4 mt-1">
                    <FormattedMessage id="component_tagger.config.query_mode_label" />
                    :
                  </Form.Label>
                  <Form.Control
                    as="select"
                    className="col-md-2 col-3 input-control"
                    value={config.mode}
                    onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
                      setConfig({
                        ...config,
                        mode: e.currentTarget.value as ParseMode,
                      })
                    }
                  >
                    <option value="auto">
                      {intl.formatMessage({
                        id: "component_tagger.config.query_mode_auto",
                      })}
                    </option>
                    <option value="filename">
                      {intl.formatMessage({
                        id: "component_tagger.config.query_mode_filename",
                      })}
                    </option>
                    <option value="dir">
                      {intl.formatMessage({
                        id: "component_tagger.config.query_mode_dir",
                      })}
                    </option>
                    <option value="path">
                      {intl.formatMessage({
                        id: "component_tagger.config.query_mode_path",
                      })}
                    </option>
                    <option value="metadata">
                      {intl.formatMessage({
                        id: "component_tagger.config.query_mode_metadata",
                      })}
                    </option>
                  </Form.Control>
                </div>
                <Form.Text>
                  {intl.formatMessage({
                    id: `component_tagger.config.query_mode_${config.mode}_desc`,
                    defaultMessage: "Unknown query mode",
                  })}
                </Form.Text>
              </Form.Group>

              <Form.Group
                controlId="stash-box-endpoint"
                className="align-items-center row no-gutters mt-4"
              >
                <Form.Label className="mr-4">
                  <FormattedMessage id="component_tagger.config.active_instance" />
                </Form.Label>
                <Form.Control
                  as="select"
                  value={config.selectedEndpoint}
                  className="col-md-4 col-6 input-control"
                  disabled={!stashBoxes.length}
                  onChange={handleInstanceSelect}
                >
                  {!stashBoxes.length && <option>No instances found</option>}
                  {stashConfig.data?.configuration.general.stashBoxes.map(
                    (i) => (
                      <option value={i.endpoint} key={i.endpoint}>
                        {i.endpoint}
                      </option>
                    )
                  )}
                </Form.Control>
              </Form.Group>
            </div>
          </div>
        </Card>
      </Collapse>
      <FieldSelector
        fields={SCENE_FIELDS}
        show={showExclusionModal}
        onSelect={handleFieldSelect}
        excludedFields={excludedFields}
      />
    </>
  );
};

export default Config;
