import { PanelPlugin } from '@grafana/data';
import { AskPanel } from './AskPanel';

interface AskPanelOptions {
  includeTimeRange?: boolean;
  includeDashboardVariables?: boolean;
  promptTemplate?: string;
}

export const plugin = new PanelPlugin<AskPanelOptions>(AskPanel).setPanelOptions(builder => {
  return builder
    .addBooleanSwitch({
      path: 'includeTimeRange',
      name: 'Include Time Range',
      description: 'Include the dashboard time range in the prompt context',
      defaultValue: true,
    })
    .addBooleanSwitch({
      path: 'includeDashboardVariables',
      name: 'Include Dashboard Variables',
      description: 'Include dashboard variables in the prompt context',
      defaultValue: true,
    })
    .addTextInput({
      path: 'promptTemplate',
      name: 'Prompt Template',
      description: 'Optional default prompt template to use',
      defaultValue: '',
      settings: {
        placeholder: 'Enter a default prompt template...',
      },
    });
});
