#!/usr/bin/env python3
import os
import sys
import yaml
import argparse
import copy

from kubernetes import client, config

def init_kube():
    """Initialize and get client"""
    config.load_kube_config()
    v1 = client.CoreV1Api()
    return v1

def shutdown_pod(v1, name, namespace):
    """Shutdown a single pod"""
    response = v1.delete_namespaced_pod(name, \
            namespace,\
            client.V1DeleteOptions(),
            grace_period_seconds=0,
            propagation_policy='Foreground')
    response = v1.delete_namespaced_service(name, \
            namespace,\
            client.V1DeleteOptions(),
            grace_period_seconds=0,
            propagation_policy='Foreground')

def find_pods(v1):
    """Find pods started by us or at least running raft-peer"""
    ret = v1.list_pod_for_all_namespaces(watch=False)
    def pod_filter(p):
        return p.metadata.namespace == "default" and \
                len(p.spec.containers) == 1 and \
                p.spec.containers[0].image == 'local-chord-node'
    pods_we_own = filter(pod_filter, ret.items)
    return pods_we_own

def kill_pod(args):
    if args.node != -1:
        v1 = init_kube()
        pods = find_pods(v1)
        peer = 'chord%d'%args.node
        pod = list(filter(lambda i: i.metadata.name == peer, pods))
        if len(pod) != 1:
            sys.exit(1)
        shutdown_pod(v1, pod[0].metadata.name, pod[0].metadata.namespace)


def kill_pods(args):
    v1 = init_kube()
    pods = find_pods(v1)
    for pod in pods:
        try:
            print("Killing:", pod.metadata.name)
            shutdown_pod(v1, pod.metadata.name, pod.metadata.namespace)
        except Exception as e:
            print("Error in killing %s %s"%(i, e), file=sys.stderr)

def boot_more(args):
    """Boot new individual pod(s)"""
    v1 = init_kube()
    pods = list(find_pods(v1))
    if len(pods) == 0:
        print("Cannot add nodes, since ring has not been initialized.\nPlease run boot command first.")
        exit()
    current_pod_count = len(pods)
    with open(os.path.join(sys.path[0], 'ring-node-template.yml')) as f:
        specs = list(yaml.load_all(f))
        peer0_node = "chord0:3001"
    for i in range(args.nodes):
        name = "chord%s"%(current_pod_count + i)
        spec_copy = copy.deepcopy(specs)
        pod_spec = spec_copy[0]
        pod_spec['metadata']['name'] = name
        pod_spec['metadata']['labels']['app'] = name
        pod_spec['spec']['containers'][0]['ports'][0]['name']="%s-client"%name
        pod_spec['spec']['containers'][0]['ports'][1]['name']="%s-chord"%name
        chord_args = ['chord']
        chord_args += ['-join', peer0_node]
        if args.piggyBack > 0:
            chord_args += ['-piggy']
        pod_spec['spec']['containers'][0]['command'] = chord_args
        service_spec =  spec_copy[1]
        # Create a service spec for this service
        service_spec['metadata']['name'] = name
        service_spec['spec']['selector']['app'] = name
        service_spec['spec']['ports'][0]['targetPort'] = "%s-client"%name
        service_spec['spec']['ports'][1]['targetPort'] = "%s-chord"%name
        try:
            response = v1.create_namespaced_pod('default', pod_spec)
            response = v1.create_namespaced_service('default', service_spec)
        except:
            print("Could not launch pod or service")
            raise

def boot(args):
    # Boot n pods
    v1 = init_kube()
    with open(os.path.join(sys.path[0], 'ring-node-template.yml')) as f:
        specs = list(yaml.load_all(f))
        peer0_node = "chord0:3001"
        for i in range(args.nodes):
            name = "chord%s"%i
            spec_copy = copy.deepcopy(specs)
            pod_spec = spec_copy[0]
            pod_spec['metadata']['name'] = name
            pod_spec['metadata']['labels']['app'] = name
            pod_spec['spec']['containers'][0]['ports'][0]['name']="%s-client"%name
            pod_spec['spec']['containers'][0]['ports'][1]['name']="%s-chord"%name
            # TODO autojoin peers
            chord_args = ['chord']
            if i > 0:
                chord_args += ['-join', peer0_node]
            if args.piggyBack > 0:
                chord_args += ['-piggy']
            pod_spec['spec']['containers'][0]['command'] = chord_args

            service_spec =  spec_copy[1]
            # Create a service spec for this service
            service_spec['metadata']['name'] = name
            service_spec['spec']['selector']['app'] = name
            service_spec['spec']['ports'][0]['targetPort'] = "%s-client"%name
            service_spec['spec']['ports'][1]['targetPort'] = "%s-chord"%name
            try:
                response = v1.create_namespaced_pod('default', pod_spec)
                response = v1.create_namespaced_service('default', service_spec)
            except:
                print("Could not launch pod or service")
                raise

def main():
    parser = argparse.ArgumentParser(prog=sys.argv[0])
    subparsers = parser.add_subparsers(help="sub-command help", dest='command')
    subparsers.required = True

    run_parser = subparsers.add_parser("boot")
    run_parser.add_argument('nodes', type=int, default=3, help='How many chord nodes?')
    run_parser.add_argument('piggyBack', type=int, default=0, help='Enable piggybacking?')
    run_parser.set_defaults(func = boot)

    run_parser = subparsers.add_parser("add")
    run_parser.add_argument('nodes', type=int, default=1, help='How many chord nodes to add to the ring?')
    run_parser.add_argument('piggyBack', type=int, default=0, help='Enable piggybacking?')
    run_parser.set_defaults(func = boot_more)

    run_parser = subparsers.add_parser("kill")
    run_parser.add_argument('node', type=int, default=-1, help='target node to kill')
    run_parser.set_defaults(func = kill_pod)

    shutdown_parser = subparsers.add_parser("shutdown")
    shutdown_parser.set_defaults(func=kill_pods)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
    sys.exit(0)
