import { infos as awsS3Infos, list as awsS3List, call as awsS3ToolCall } from './aws-s3';
import { infos as awsSESInfos, list as awsSESLList, call as awsSESToolCall } from './aws-ses';
import { infos as beamlitSearchInfos, list as beamlitSearchList, call as beamlitSearchToolCall } from './beamlit-search';
import { infos as braveInfos, list as braveList, call as braveToolCall } from './brave';
import { infos as cloudflareInfos, list as cloudflareList, call as cloudflareToolCall } from './cloudflare';
import { infos as dalleInfos, list as dalleList, call as dalleToolCall } from './dall-e';
import { infos as githubInfos, list as githubList, call as githubToolCall } from './github';
import { infos as gmailInfos, list as gmailList, call as gmailToolCall } from './gmail';
import { infos as gmapInfos, list as gmapList, call as gmapToolCall } from './google-maps';
import { infos as linearInfos, list as linearList, call as linearToolCall } from './linear';
import { infos as postgresInfos, list as postgresList, call as postgresToolCall } from './postgres';
import { infos as qdrantInfos, list as qdrantList, call as qdrantToolCall } from './qdrant';
import { infos as slackInfos, list as slackList, call as slackToolCall } from './slack';

export const mcpServers: Record<string, Record<string, Function>> = {
	'beamlit-search': {
		list: beamlitSearchList,
		call: beamlitSearchToolCall,
		infos: beamlitSearchInfos,
	},
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
	cloudflare: {
		list: cloudflareList,
		call: cloudflareToolCall,
		infos: cloudflareInfos,
	},
	postgres: {
		list: postgresList,
		call: postgresToolCall,
		infos: postgresInfos,
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
};
