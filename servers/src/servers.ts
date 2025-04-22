import { infos as awsS3Infos, list as awsS3List, call as awsS3ToolCall } from './aws-s3/index.js';
import { infos as awsSESInfos, list as awsSESLList, call as awsSESToolCall } from './aws-ses/index.js';
import { infos as braveInfos, list as braveList, call as braveToolCall } from './brave-search/index.js';
import { infos as cloudflareInfos, list as cloudflareList, call as cloudflareToolCall } from './cloudflare/index.js';
import { infos as dalleInfos, list as dalleList, call as dalleToolCall } from './dall-e/index.js';
import { infos as githubInfos, list as githubList, call as githubToolCall } from './github/index.js';
import { infos as gmailInfos, list as gmailList, call as gmailToolCall } from './gmail/index.js';
import { infos as googleDriveInfos, list as googleDriveList, call as googleDriveToolCall } from './google-drive/index.js';
import { infos as gmapInfos, list as gmapList, call as gmapToolCall } from './google-maps/index.js';
import { infos as linearInfos, list as linearList, call as linearToolCall } from './linear/index.js';
import { infos as qdrantInfos, list as qdrantList, call as qdrantToolCall } from './qdrant/index.js';
import { infos as slackInfos, list as slackList, call as slackToolCall } from './slack/index.js';
import { infos as twilioInfos, list as twilioList, call as twilioToolCall } from './twilio/index.js';

export const mcpServers: Record<string, Record<string, Function>> = {
	'brave-search': {
		list: braveList,
		call: braveToolCall,
		infos: braveInfos,
	},
	github: {
		list: githubList,
		call: githubToolCall,
		infos: githubInfos,
	},
	slack: {
		list: slackList,
		call: slackToolCall,
		infos: slackInfos,
	},
	'google-maps': {
		list: gmapList,
		call: gmapToolCall,
		infos: gmapInfos,
	},
	gmail: {
		list: gmailList,
		call: gmailToolCall,
		infos: gmailInfos,
	},
	'google-drive': {
		list: googleDriveList,
		call: googleDriveToolCall,
		infos: googleDriveInfos,
	},
	cloudflare: {
		list: cloudflareList,
		call: cloudflareToolCall,
		infos: cloudflareInfos,
	},
	'aws-s3': {
		list: awsS3List,
		call: awsS3ToolCall,
		infos: awsS3Infos,
	},
	'aws-ses': {
		list: awsSESLList,
		call: awsSESToolCall,
		infos: awsSESInfos,
	},
	'dall-e': {
		list: dalleList,
		call: dalleToolCall,
		infos: dalleInfos,
	},
	linear: {
		list: linearList,
		call: linearToolCall,
		infos: linearInfos,
	},
	qdrant: {
		list: qdrantList,
		call: qdrantToolCall,
		infos: qdrantInfos,
	},
	twilio: {
		list: twilioList,
		call: twilioToolCall,
		infos: twilioInfos,
	},
};
