import React, { useRef } from "react";
import { Form } from "react-bootstrap";
import { CriterionModifier } from "../../../core/generated-graphql";
import { INumberValue } from "../../../models/list-filter/types";
import { Criterion } from "../../../models/list-filter/criteria/criterion";

interface IDurationFilterProps {
  criterion: Criterion<INumberValue>;
  onValueChanged: (value: INumberValue) => void;
}

export const NumberFilter: React.FC<IDurationFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  const valueStage = useRef<INumberValue>(criterion.value);

  function onChanged(
    event: React.ChangeEvent<HTMLInputElement>,
    property: "exact" | "lower" | "upper"
  ) {
    valueStage.current[property] = parseInt(event.target.value, 10);
  }

  function onBlurInput() {
    onValueChanged(valueStage.current);
  }

  if (
    criterion.modifier === CriterionModifier.Equals ||
    criterion.modifier === CriterionModifier.NotEquals
  ) {
    return (
      <Form.Group>
        <Form.Control
          className="btn-secondary"
          type="number"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e, "exact")
          }
          onBlur={onBlurInput}
          defaultValue={criterion.value?.exact ?? ""}
        />
      </Form.Group>
    );
  }

  let lowerControl: JSX.Element | null = null;
  if (
    criterion.modifier === CriterionModifier.GreaterThan ||
    criterion.modifier === CriterionModifier.Between ||
    criterion.modifier === CriterionModifier.NotBetween
  ) {
    lowerControl = (
      <Form.Group>
        <Form.Control
          className="btn-secondary"
          type="number"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e, "lower")
          }
          onBlur={onBlurInput}
          defaultValue={criterion.value?.lower ?? ""}
        />
      </Form.Group>
    );
  }

  let upperControl: JSX.Element | null = null;
  if (
    criterion.modifier === CriterionModifier.LessThan ||
    criterion.modifier === CriterionModifier.Between ||
    criterion.modifier === CriterionModifier.NotBetween
  ) {
    upperControl = (
      <Form.Group>
        <Form.Control
          className="btn-secondary"
          type="number"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e, "upper")
          }
          onBlur={onBlurInput}
          defaultValue={criterion.value?.upper ?? ""}
        />
      </Form.Group>
    );
  }

  return (
    <>
      {lowerControl}
      {upperControl}
    </>
  );
};
