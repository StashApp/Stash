import React from "react";
import { Form } from "react-bootstrap";
import { CriterionModifier } from "../../../core/generated-graphql";
import { DurationInput } from "../../Shared";
import { INumberValue } from "../../../models/list-filter/types";
import { Criterion } from "../../../models/list-filter/criteria/criterion";

interface IDurationFilterProps {
  criterion: Criterion<INumberValue>;
  onValueChanged: (value: INumberValue) => void;
}

export const DurationFilter: React.FC<IDurationFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  function onChanged(
    valueAsNumber: number,
    property: "exact" | "lower" | "upper"
  ) {
    const { value } = criterion;
    value[property] = valueAsNumber;
    onValueChanged(value);
  }

  if (
    criterion.modifier === CriterionModifier.Equals ||
    criterion.modifier === CriterionModifier.NotEquals
  ) {
    return (
      <Form.Group>
        <DurationInput
          numericValue={criterion.value?.exact ? criterion.value.exact : 0}
          onValueChange={(v: number) => onChanged(v, "exact")}
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
        <DurationInput
          numericValue={criterion.value?.lower ? criterion.value.lower : 0}
          onValueChange={(v: number) => onChanged(v, "lower")}
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
        <DurationInput
          numericValue={criterion.value?.upper ? criterion.value.upper : 0}
          onValueChange={(v: number) => onChanged(v, "upper")}
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
