package systemd

// copied from virtual kubelet zun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/miekg/vks/pkg/unit"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// GetPod returns ...
func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.Print("GetPod called")
	units, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	unitpref := UnitPrefix(namespace, name)
	for name, _ := range units {
		if !strings.HasPrefix(name, unitpref) {
			delete(units, name)
		}
	}
	pod := unitToPod(units)
	return pod, nil
}

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	states, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	// sort unit by namespace/name
	for name, s := range states {
		fmt.Printf("GETPODS: %s: %+v\n", name, s)
	}
	return nil, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Print("CreatedPod called")
	// Can we store metadata somewhere within systemd units files?
	/*
		metadata.Labels = map[string]string{
			"PodName":           pod.Name,
			"ClusterName":       pod.ClusterName,
			"NodeName":          pod.Spec.NodeName,
			"Namespace":         pod.Namespace,
			"UID":               podUID,
			"CreationTimestamp": podCreationTimestamp,
		}
	*/
	for _, c := range pod.Spec.Containers {
		// parse c.Image for tag?
		if err := p.pkg.Install(c.Image, ""); err != nil {
			return err
		}
		u, err := p.pkg.Unitfile(c.Image)
		if err != nil {
			return err
		}
		name := PodToUnitName(pod, c.Name)
		log.Printf("Starting unit %s, %s as %s", c.Name, c.Image, name)
		buf, err := ioutil.ReadFile(u)
		if err != nil {
			return err
		}
		uf, err := unit.NewUnitFile(string(buf))
		if err != nil {
			return err
		}
		if err := p.m.Load(name, *uf); err != nil {
			return err
		}
		if err := p.m.TriggerStart(name); err != nil {
			return err
		}

	}
	return nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *P) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	// technically we can just give you a system shell here....
	log.Printf("receive RunInContainer %q\n", container)
	return nil
}

// GetPodStatus returns the status of a pod by name that is running.
// returns nil if a pod by that name is not found.
func (p *P) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.Printf("GetPodStatus called")
	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		return nil, nil
	}
	return &pod.Status, nil
}

func (p *P) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	log.Printf("GetContainerLogs called")

	args := []string{"-u", NamespaceNameToUnitName(namespace, podName, containerName, "*")} // * to wildcard the unit name (we don't have the uid)
	cmd := exec.Command("journalctl", args...)
	// returns the buffers? What about following, use pipes here or something?
	buf, err := cmd.CombinedOutput()
	return ioutil.NopCloser(bytes.NewReader(buf)), err
}

// UpdatePod is a noop,
func (p *P) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("UpdatePod called")
	return nil
}

// DeletePod deletes
func (p *P) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("DeletePod called")
	for _, c := range pod.Spec.Containers {
		name := PodToUnitName(pod, c.Name)
		if err := p.m.TriggerStop(name); err != nil {
			log.Printf("Failed to triggger top: %s", err)
		}
		if err := p.m.Unload(name); err != nil {
			log.Printf("Failed to unload: %s", err)
		}
	}
	return nil
}

func PodToUnitName(pod *corev1.Pod, image string) string {
	return NamespaceNameToUnitName(pod.Namespace, pod.Name, image, string(pod.UID))
}

// We use image here, but it should containerName (probably)
func NamespaceNameToUnitName(namespace, name, image, uid string) string {
	return UnitPrefix(namespace, name) + Separator + image + Separator + uid + ".service"
}

func UnitPrefix(namespace, name string) string {
	return Prefix + Separator + namespace + Separator + name
}

func Image(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[3]
}

func Name(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[1] + Separator + el[2]
}

func Pod(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[2]
}

func UID(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 5 {
		return ""
	}
	return el[4]
}

func Namespace(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[1]
}

const (
	Prefix    = "vks"
	Separator = "."
)
