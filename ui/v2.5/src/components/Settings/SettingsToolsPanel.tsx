import React from "react";
import { Form } from "react-bootstrap";
import { Link } from "react-router-dom";

export const SettingsToolsPanel: React.FC = () => {
  return (
    <>
      <h4>Scene Tools</h4>

      <Button
        variant="secondary"
        type="submit"
      >
        <a href-"/sceneFilenameParser">Scene Filename Parser</a>
      </Button>
      <Button
        variant="secondary"
        type="submit"
      >
        <a href-"/sceneDuplicateChecker">Scene Duplicate Checker</a>
      </Button>
    </>
  );
};
