import React, { useEffect, useState } from "react";
import { Button, Form } from "react-bootstrap";
import { DurationInput, LoadingIndicator } from "src/components/Shared";
import { useConfiguration, useConfigureInterface } from "src/core/StashService";
import { useToast } from "src/hooks";
import { CheckboxGroup } from "./CheckboxGroup";

const allMenuItems = [
  { id: "scenes", label: "Scenes" },
  { id: "images", label: "Images" },
  { id: "movies", label: "Movies" },
  { id: "markers", label: "Markers" },
  { id: "galleries", label: "Galleries" },
  { id: "performers", label: "Performers" },
  { id: "studios", label: "Studios" },
  { id: "tags", label: "Tags" },
];

const SECONDS_TO_MS = 1000;

export const SettingsInterfacePanel: React.FC = () => {
  const Toast = useToast();
  const { data: config, error, loading } = useConfiguration();
  const [menuItemIds, setMenuItemIds] = useState<string[]>(
    allMenuItems.map((item) => item.id)
  );
  const [soundOnPreview, setSoundOnPreview] = useState<boolean>(true);
  const [wallShowTitle, setWallShowTitle] = useState<boolean>(true);
  const [wallPlayback, setWallPlayback] = useState<string>("video");
  const [maximumLoopDuration, setMaximumLoopDuration] = useState<number>(0);
  const [autostartVideo, setAutostartVideo] = useState<boolean>(false);
  const [slideshowDelay, setSlideshowDelay] = useState<number>(0);
  const [showStudioAsText, setShowStudioAsText] = useState<boolean>(false);
  const [css, setCSS] = useState<string>();
  const [cssEnabled, setCSSEnabled] = useState<boolean>(false);
  const [language, setLanguage] = useState<string>("en");
  const [handyKey, setHandyKey] = useState<string>();

  const [updateInterfaceConfig] = useConfigureInterface({
    menuItems: menuItemIds,
    soundOnPreview,
    wallShowTitle,
    wallPlayback,
    maximumLoopDuration,
    autostartVideo,
    showStudioAsText,
    css,
    cssEnabled,
    language,
    slideshowDelay,
    handyKey,
  });

  useEffect(() => {
    const iCfg = config?.configuration?.interface;
    setMenuItemIds(iCfg?.menuItems ?? allMenuItems.map((item) => item.id));
    setSoundOnPreview(iCfg?.soundOnPreview ?? true);
    setWallShowTitle(iCfg?.wallShowTitle ?? true);
    setWallPlayback(iCfg?.wallPlayback ?? "video");
    setMaximumLoopDuration(iCfg?.maximumLoopDuration ?? 0);
    setAutostartVideo(iCfg?.autostartVideo ?? false);
    setShowStudioAsText(iCfg?.showStudioAsText ?? false);
    setCSS(iCfg?.css ?? "");
    setCSSEnabled(iCfg?.cssEnabled ?? false);
    setLanguage(iCfg?.language ?? "en-US");
    setSlideshowDelay(iCfg?.slideshowDelay ?? 5000);
    setHandyKey(iCfg?.handyKey ?? "");
  }, [config]);

  async function onSave() {
    const prevCSS = config?.configuration.interface.css;
    const prevCSSenabled = config?.configuration.interface.cssEnabled;
    try {
      const result = await updateInterfaceConfig();
      // eslint-disable-next-line no-console
      console.log(result);

      // Force refetch of custom css if it was changed
      if (
        prevCSS !== result.data?.configureInterface.css ||
        prevCSSenabled !== result.data?.configureInterface.cssEnabled
      ) {
        await fetch("/css", { cache: "reload" });
        window.location.reload();
      }

      Toast.success({ content: "Updated config" });
    } catch (e) {
      Toast.error(e);
    }
  }

  if (error) return <h1>{error.message}</h1>;
  if (loading) return <LoadingIndicator />;

  return (
    <>
      <h4>User Interface</h4>
      <Form.Group controlId="language">
        <h5>Language</h5>
        <Form.Control
          as="select"
          className="col-4 input-control"
          value={language}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            setLanguage(e.currentTarget.value)
          }
        >
          <option value="en-US">English (United States)</option>
          <option value="en-GB">English (United Kingdom)</option>
          <option value="zh-TW">Chinese (Taiwan)</option>
          <option value="zh-CN">Chinese (Simplified)</option>
        </Form.Control>
      </Form.Group>
      <Form.Group>
        <h5>Menu items</h5>
        <CheckboxGroup
          groupId="menu-items"
          items={allMenuItems}
          checkedIds={menuItemIds}
          onChange={setMenuItemIds}
        />
        <Form.Text className="text-muted">
          Show or hide different types of content on the navigation bar
        </Form.Text>
      </Form.Group>
      <Form.Group>
        <h5>Scene / Marker Wall</h5>
        <Form.Check
          id="wall-show-title"
          checked={wallShowTitle}
          label="Display title and tags"
          onChange={() => setWallShowTitle(!wallShowTitle)}
        />
        <Form.Check
          id="wall-sound-enabled"
          checked={soundOnPreview}
          label="Enable sound"
          onChange={() => setSoundOnPreview(!soundOnPreview)}
        />
        <Form.Label htmlFor="wall-preview">
          <h6>Preview Type</h6>
        </Form.Label>
        <Form.Control
          as="select"
          name="wall-preview"
          className="col-4 input-control"
          value={wallPlayback}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            setWallPlayback(e.currentTarget.value)
          }
        >
          <option value="video">Video</option>
          <option value="animation">Animated Image</option>
          <option value="image">Static Image</option>
        </Form.Control>
        <Form.Text className="text-muted">
          Configuration for wall items
        </Form.Text>
      </Form.Group>

      <Form.Group>
        <h5>Scene List</h5>
        <Form.Check
          id="show-text-studios"
          checked={showStudioAsText}
          label="Show Studios as text"
          onChange={() => {
            setShowStudioAsText(!showStudioAsText);
          }}
        />
      </Form.Group>

      <Form.Group>
        <h5>Scene Player</h5>
        <Form.Group id="auto-start-video">
          <Form.Check
            checked={autostartVideo}
            label="Auto-start video"
            onChange={() => {
              setAutostartVideo(!autostartVideo);
            }}
          />
        </Form.Group>

        <Form.Group id="max-loop-duration">
          <h6>Maximum loop duration</h6>
          <DurationInput
            className="row col col-4"
            numericValue={maximumLoopDuration}
            onValueChange={(duration) => setMaximumLoopDuration(duration ?? 0)}
          />
          <Form.Text className="text-muted">
            Maximum scene duration where scene player will loop the video - 0 to
            disable
          </Form.Text>
        </Form.Group>
      </Form.Group>

      <Form.Group id="slideshow-delay">
        <h5>Slideshow Delay</h5>
        <Form.Control
          className="col col-sm-6 text-input"
          type="number"
          value={slideshowDelay / SECONDS_TO_MS}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            setSlideshowDelay(
              Number.parseInt(e.currentTarget.value, 10) * SECONDS_TO_MS
            );
          }}
        />
        <Form.Text className="text-muted">
          Slideshow is available in galleries when in wall view mode
        </Form.Text>
      </Form.Group>

      <Form.Group>
        <h5>Custom CSS</h5>
        <Form.Check
          id="custom-css"
          checked={cssEnabled}
          label="Custom CSS enabled"
          onChange={() => {
            setCSSEnabled(!cssEnabled);
          }}
        />

        <Form.Control
          as="textarea"
          value={css}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
            setCSS(e.currentTarget.value)
          }
          rows={16}
          className="col col-sm-6 text-input code"
        />
        <Form.Text className="text-muted">
          Page must be reloaded for changes to take effect.
        </Form.Text>
      </Form.Group>

      <Form.Group>
        <h5>Handy Connection Key</h5>
        <Form.Control
          className="col col-sm-6 text-input"
          value={handyKey}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            setHandyKey(e.currentTarget.value);
          }}
        />
        <Form.Text className="text-muted">
          Handy connection key to use for interactive scenes.
        </Form.Text>
      </Form.Group>

      <hr />
      <Button variant="primary" onClick={() => onSave()}>
        Save
      </Button>
    </>
  );
};
