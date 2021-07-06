/* eslint-disable consistent-return */

import { IntlShape } from "react-intl";
import {
  CriterionModifier,
  HierarchicalMultiCriterionInput,
  IntCriterionInput,
  MultiCriterionInput,
} from "src/core/generated-graphql";
import DurationUtils from "src/utils/duration";
import {
  CriterionType,
  encodeILabeledId,
  IHierarchicalLabelValue,
  ILabeledId,
  ILabeledValue,
  INumberValue,
  IOptionType,
} from "../types";

export type Option = string | number | IOptionType;
export type CriterionValue =
  | string
  | ILabeledId[]
  | IHierarchicalLabelValue
  | INumberValue;

const modifierMessageIDs = {
  [CriterionModifier.Equals]: "criterion_modifier.equals",
  [CriterionModifier.NotEquals]: "criterion_modifier.not_equals",
  [CriterionModifier.GreaterThan]: "criterion_modifier.greater_than",
  [CriterionModifier.LessThan]: "criterion_modifier.less_than",
  [CriterionModifier.IsNull]: "criterion_modifier.is_null",
  [CriterionModifier.NotNull]: "criterion_modifier.not_null",
  [CriterionModifier.Includes]: "criterion_modifier.includes",
  [CriterionModifier.IncludesAll]: "criterion_modifier.includes_all",
  [CriterionModifier.Excludes]: "criterion_modifier.excludes",
  [CriterionModifier.MatchesRegex]: "criterion_modifier.matches_regex",
  [CriterionModifier.NotMatchesRegex]: "criterion_modifier.not_matches_regex",
  [CriterionModifier.Between]: "criterion_modifier.between",
  [CriterionModifier.NotBetween]: "criterion_modifier.not_between",
};

// V = criterion value type
export abstract class Criterion<V extends CriterionValue> {
  public static getModifierOption(
    modifier: CriterionModifier = CriterionModifier.Equals
  ): ILabeledValue {
    const messageID = modifierMessageIDs[modifier];
    return { value: modifier, label: messageID };
  }

  public criterionOption: CriterionOption;
  public modifier: CriterionModifier;
  public value: V;

  public abstract getLabelValue(): string;

  constructor(type: CriterionOption, value: V) {
    this.criterionOption = type;
    this.modifier = type.defaultModifier;
    this.value = value;
  }

  public static getModifierLabel(intl: IntlShape, modifier: CriterionModifier) {
    const modifierMessageID = modifierMessageIDs[modifier];

    return modifierMessageID
      ? intl.formatMessage({ id: modifierMessageID })
      : "";
  }

  public getLabel(intl: IntlShape): string {
    const modifierString = Criterion.getModifierLabel(intl, this.modifier);
    let valueString = "";

    if (
      this.modifier !== CriterionModifier.IsNull &&
      this.modifier !== CriterionModifier.NotNull
    ) {
      valueString = this.getLabelValue();
    }

    return intl.formatMessage(
      { id: "criterion_modifier.format_string" },
      {
        criterion: intl.formatMessage({ id: this.criterionOption.messageID }),
        modifierString,
        valueString,
      }
    );
  }

  public getId(): string {
    return `${this.criterionOption.parameterName}-${this.modifier.toString()}`; // TODO add values?
  }

  public encodeValue(): V {
    return this.value;
  }

  public toJSON() {
    const encodedCriterion = {
      type: this.criterionOption.type,
      // #394 - the presence of a # symbol results in the query URL being
      // malformed. We could set encode: true in the queryString.stringify
      // call below, but this results in a URL that gets pretty long and ugly.
      // Instead, we'll encode the criteria values.
      value: this.encodeValue(),
      modifier: this.modifier,
    };
    return JSON.stringify(encodedCriterion);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  public apply(outputFilter: Record<string, any>) {
    // eslint-disable-next-line no-param-reassign
    outputFilter[this.criterionOption.parameterName] = this.toCriterionInput();
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  protected toCriterionInput(): any {
    return {
      value: this.value,
      modifier: this.modifier,
    };
  }
}

export type InputType = "number" | "text" | undefined;

interface ICriterionOptionsParams {
  messageID: string;
  type: CriterionType;
  inputType?: InputType;
  parameterName?: string;
  modifierOptions?: CriterionModifier[];
  defaultModifier?: CriterionModifier;
  options?: Option[];
}
export class CriterionOption {
  public readonly messageID: string;
  public readonly type: CriterionType;
  public readonly parameterName: string;
  public readonly modifierOptions: ILabeledValue[];
  public readonly defaultModifier: CriterionModifier;
  public readonly options: Option[] | undefined;
  public readonly inputType: InputType;

  constructor(options: ICriterionOptionsParams) {
    this.messageID = options.messageID;
    this.type = options.type;
    this.parameterName = options.parameterName ?? options.type;
    this.modifierOptions = (options.modifierOptions ?? []).map((o) =>
      Criterion.getModifierOption(o)
    );
    this.defaultModifier = options.defaultModifier ?? CriterionModifier.Equals;
    this.options = options.options;
    this.inputType = options.inputType;
  }
}

export class StringCriterionOption extends CriterionOption {
  constructor(
    messageID: string,
    value: CriterionType,
    parameterName?: string,
    options?: Option[]
  ) {
    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.Includes,
        CriterionModifier.Excludes,
        CriterionModifier.IsNull,
        CriterionModifier.NotNull,
        CriterionModifier.MatchesRegex,
        CriterionModifier.NotMatchesRegex,
      ],
      defaultModifier: CriterionModifier.Equals,
      options,
      inputType: "text",
    });
  }
}

export function createStringCriterionOption(
  value: CriterionType,
  messageID?: string,
  parameterName?: string
) {
  return new StringCriterionOption(
    messageID ?? value,
    value,
    parameterName ?? messageID ?? value
  );
}

export class StringCriterion extends Criterion<string> {
  public getLabelValue() {
    return this.value;
  }

  public encodeValue() {
    // replace certain characters
    let ret = this.value;
    ret = StringCriterion.replaceSpecialCharacter(ret, "&");
    ret = StringCriterion.replaceSpecialCharacter(ret, "+");
    return ret;
  }

  private static replaceSpecialCharacter(str: string, c: string) {
    return str.replaceAll(c, encodeURIComponent(c));
  }

  constructor(type: CriterionOption) {
    super(type, "");
  }
}

export class MandatoryStringCriterionOption extends CriterionOption {
  constructor(
    messageID: string,
    value: CriterionType,
    parameterName?: string,
    options?: Option[]
  ) {
    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.Includes,
        CriterionModifier.Excludes,
        CriterionModifier.MatchesRegex,
        CriterionModifier.NotMatchesRegex,
      ],
      defaultModifier: CriterionModifier.Equals,
      options,
      inputType: "text",
    });
  }
}

export function createMandatoryStringCriterionOption(
  value: CriterionType,
  messageID?: string,
  parameterName?: string
) {
  return new MandatoryStringCriterionOption(
    messageID ?? value,
    value,
    parameterName ?? messageID ?? value
  );
}

export class BooleanCriterionOption extends CriterionOption {
  constructor(messageID: string, value: CriterionType, parameterName?: string) {
    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions: [],
      defaultModifier: CriterionModifier.Equals,
      options: [true.toString(), false.toString()],
    });
  }
}

export class BooleanCriterion extends StringCriterion {
  protected toCriterionInput(): boolean {
    return this.value === "true";
  }
}

export class NumberCriterionOption extends CriterionOption {
  constructor(
    messageID: string,
    value: CriterionType,
    parameterName?: string,
    options?: Option[]
  ) {
    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.GreaterThan,
        CriterionModifier.LessThan,
        CriterionModifier.IsNull,
        CriterionModifier.NotNull,
        CriterionModifier.Between,
        CriterionModifier.NotBetween,
      ],
      defaultModifier: CriterionModifier.Equals,
      options,
      inputType: "number",
    });
  }
}

export function createNumberCriterionOption(value: CriterionType) {
  return new NumberCriterionOption(value, value, value);
}

export class NumberCriterion extends Criterion<INumberValue> {
  public encodeValue() {
    return {
      exact: this.value.exact,
      lower: this.value.lower,
      upper: this.value.upper,
    };
  }

  protected toCriterionInput(): IntCriterionInput {
    return {
      modifier: this.modifier,
      exact: this.value.exact,
      lower: this.value.lower,
      upper: this.value.upper,
    };
  }

  public getLabelValue() {
    return this.modifier === CriterionModifier.Between ||
      this.modifier === CriterionModifier.NotBetween
      ? `${this.value.lower}, ${this.value.upper}`
      : this.modifier === CriterionModifier.GreaterThan
      ? `${this.value.lower}`
      : this.modifier === CriterionModifier.LessThan
      ? `${this.value.upper}`
      : `${this.value.exact}`;
  }

  constructor(type: CriterionOption) {
    super(type, { exact: 0, lower: 0, upper: 0 });
  }
}

export class ILabeledIdCriterionOption extends CriterionOption {
  constructor(
    messageID: string,
    value: CriterionType,
    parameterName: string,
    includeAll: boolean
  ) {
    const modifierOptions = [
      CriterionModifier.Includes,
      CriterionModifier.Excludes,
    ];

    let defaultModifier = CriterionModifier.Includes;
    if (includeAll) {
      modifierOptions.unshift(CriterionModifier.IncludesAll);
      defaultModifier = CriterionModifier.IncludesAll;
    }

    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions,
      defaultModifier,
    });
  }
}

export class ILabeledIdCriterion extends Criterion<ILabeledId[]> {
  public getLabelValue(): string {
    return this.value.map((v) => v.label).join(", ");
  }

  protected toCriterionInput(): MultiCriterionInput {
    return {
      value: this.value.map((v) => v.id),
      modifier: this.modifier,
    };
  }

  public encodeValue() {
    return this.value.map((o) => {
      return encodeILabeledId(o);
    });
  }

  constructor(type: CriterionOption) {
    super(type, []);
  }
}

export class IHierarchicalLabeledIdCriterion extends Criterion<IHierarchicalLabelValue> {
  public encodeValue() {
    return {
      items: this.value.items.map((o) => {
        return encodeILabeledId(o);
      }),
      depth: this.value.depth,
    };
  }

  protected toCriterionInput(): HierarchicalMultiCriterionInput {
    return {
      value: (this.value.items ?? []).map((v) => v.id),
      modifier: this.modifier,
      depth: this.value.depth,
    };
  }

  public getLabelValue(): string {
    const labels = (this.value.items ?? []).map((v) => v.label).join(", ");

    if (this.value.depth === 0) {
      return labels;
    }

    return `${labels} (+${this.value.depth > 0 ? this.value.depth : "all"})`;
  }

  constructor(type: CriterionOption) {
    const value: IHierarchicalLabelValue = {
      items: [],
      depth: 0,
    };

    super(type, value);
  }
}

export class MandatoryNumberCriterionOption extends CriterionOption {
  constructor(messageID: string, value: CriterionType, parameterName?: string) {
    super({
      messageID,
      type: value,
      parameterName,
      modifierOptions: [
        CriterionModifier.Equals,
        CriterionModifier.NotEquals,
        CriterionModifier.GreaterThan,
        CriterionModifier.LessThan,
        CriterionModifier.Between,
        CriterionModifier.NotBetween,
      ],
      defaultModifier: CriterionModifier.Equals,
      inputType: "number",
    });
  }
}

export function createMandatoryNumberCriterionOption(value: CriterionType) {
  return new MandatoryNumberCriterionOption(value, value, value);
}

export class DurationCriterion extends Criterion<INumberValue> {
  constructor(type: CriterionOption) {
    super(type, { exact: 0, lower: 0, upper: 0 });
  }

  public encodeValue() {
    return {
      exact: this.value.exact,
      lower: this.value.lower,
      upper: this.value.upper,
    };
  }

  protected toCriterionInput(): IntCriterionInput {
    return {
      modifier: this.modifier,
      exact: this.value.exact,
      lower: this.value.lower,
      upper: this.value.upper,
    };
  }

  public getLabelValue() {
    return this.modifier === CriterionModifier.Between ||
      this.modifier === CriterionModifier.NotBetween
      ? `${DurationUtils.secondsToString(
          this.value.lower
        )} ${DurationUtils.secondsToString(this.value.upper)}`
      : this.modifier === CriterionModifier.GreaterThan
      ? DurationUtils.secondsToString(this.value.lower)
      : this.modifier === CriterionModifier.LessThan
      ? DurationUtils.secondsToString(this.value.upper)
      : this.modifier === CriterionModifier.Equals ||
        this.modifier === CriterionModifier.NotEquals
      ? DurationUtils.secondsToString(this.value.exact)
      : "?";
  }
}
